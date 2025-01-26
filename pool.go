package routinepool

import (
	"sync"
	"sync/atomic"
	"time"
)

// Pool worker routine pool
type Pool struct {
	mu sync.Mutex
	wg sync.WaitGroup

	maxWorkers     int   // max worker routine numbers
	runningWorkers int64 // in running worker numbers
	idleWorkers    int64 // idle worker routine numbers
	taskProcessed  int64
	started        bool

	workerStopChan chan struct{}
	controlChan    chan signal
	taskChan       chan TaskFunc
}

// TaskFunc defines a task function
type TaskFunc func()

// worker idle timeout to exit
const WorkerIdleTimeoutToExit = time.Second

// signal type
type signal int

const (
	signalWorkerRoutinePanic signal = iota
	signalStopWorkers
	signalAddTask
)

// New create a routine pool with options.
func New(opts ...Option) *Pool {
	o := defaultOptions()
	for _, option := range opts {
		option.apply(o)
	}
	pool := &Pool{
		maxWorkers:  o.maxWorkers,
		controlChan: make(chan signal, 8),
		taskChan:    make(chan TaskFunc, o.maxTaskSize),
	}

	return pool
}

// Start start routine pool in background.
// The initial number of workers is determined by the number of tasks in the channel and maxWorkers.
func (p *Pool) Start() {
	p.start(min(len(p.taskChan), p.maxWorkers))
}

// StartN start routine pool in background with workerNum initial workers.
// if workerNum <= 0 or workerNum > maxWorkers, it will be set to maxWorkers.
func (p *Pool) StartN(workerNum int) {
	if workerNum < 0 || workerNum > p.maxWorkers {
		workerNum = p.maxWorkers
	}
	p.start(workerNum)
}

func (p *Pool) start(workerNum int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	p.workerStopChan = make(chan struct{})
	for i := 0; i < workerNum; i++ {
		p.startWorkerRoutine()
	}

	p.wg.Add(1)
	go p.startControllerRoutine()
	p.started = true
}

// start controller routine
func (p *Pool) startControllerRoutine() {
	defer p.wg.Done()
	defer func() {
		p.started = false
	}()

	stop := false

	for {
		if stop && atomic.LoadInt64(&p.runningWorkers) == 0 {
			return
		}

		s := <-p.controlChan
		switch s {
		case signalWorkerRoutinePanic:
			if !stop {
				p.startWorkerRoutine()
			}
		case signalStopWorkers:
			stop = true
			close(p.workerStopChan)
		case signalAddTask:
			if !stop && atomic.LoadInt64(&p.idleWorkers) == 0 && atomic.LoadInt64(&p.runningWorkers) < int64(p.maxWorkers) {
				p.startWorkerRoutine()
			}
		}
	}
}

// start worker routine
func (p *Pool) startWorkerRoutine() {
	atomic.AddInt64(&p.runningWorkers, 1)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer func() {
			atomic.AddInt64(&p.runningWorkers, -1)

			if r := recover(); r != nil {
				p.controlChan <- signalWorkerRoutinePanic
			}
		}()

		idleTimer := time.NewTimer(WorkerIdleTimeoutToExit)
		atomic.AddInt64(&p.idleWorkers, 1)

		for {
			resetTimer(idleTimer, WorkerIdleTimeoutToExit)

			select {
			case <-p.workerStopChan:
				return
			case task := <-p.taskChan:
				atomic.AddInt64(&p.idleWorkers, -1)
				task()
				atomic.AddInt64(&p.taskProcessed, 1)
				atomic.AddInt64(&p.idleWorkers, 1)
			case <-idleTimer.C:
				return
			}
		}
	}()
}

// resetTimer stops, drains and resets the timer.
// there is a Timer.Reset issue: https://github.com/golang/go/issues/11513
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// AddTask add a task to Pool, the worker routine will execute the task.
// if task channel is full, it will be blocked.
func (p *Pool) AddTask(task TaskFunc) {
	if task == nil {
		return
	}
	p.controlChan <- signalAddTask
	p.taskChan <- task
}

// Stop stop all routines no matter how many tasks wait for execution.
func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// send stop all routines signal to control routine
	p.controlChan <- signalStopWorkers
	// waite
	p.wg.Wait()
}

// Wait for blocking, and wait for all tasks to be executed and no tasks to be
// executed, then stop the pool.
// if taskChan never empty, it wouldn't return.
func (p *Pool) Wait() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	for {
		// waiting for all tasks be executed and no running workers
		if len(p.taskChan) == 0 && atomic.LoadInt64(&p.runningWorkers) == 0 {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// WaitIdleTimeout waiting with idle timeout, then stop the pool.
// if taskChan never empty, it wouldn't return.
func (p *Pool) WaitIdleTimeout(idleTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	allIdle := false
	for {
		// waiting for all tasks be executed and no running workers
		if len(p.taskChan) == 0 && atomic.LoadInt64(&p.runningWorkers) == 0 {
			if allIdle {
				return
			}
			allIdle = true
			time.Sleep(idleTimeout)
			continue
		} else {
			allIdle = false
		}
		time.Sleep(time.Millisecond * 100)
	}
}
