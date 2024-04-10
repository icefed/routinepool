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

	maxWorkers     int               // max worker routine numbers
	runningWorkers map[int64]*worker // in running worker routines
	idleWorkers    int64             // idle worker routine numbers
	idleTimeout    time.Duration
	workerSeq      int64
	started        bool

	controlChan chan signal
	taskChan    chan TaskFunc
}

type worker struct {
	id   int64
	wait chan time.Duration
}

// TaskFunc defines a task function
type TaskFunc func()

type signal struct {
	signal int
	// signal from worker's id
	workerId int64
	// call Wait() with worker idle timeout, send to workers
	idleTimeout time.Duration
}

// signal type
const (
	signalWorkerRoutineExit = iota
	signalWorkerRoutinePanic
	signalWaiteWorkers
	signalStopWorkers
	signalAddTask
)

// NewPool create a routine pool with options.
func NewPool(opts ...Option) *Pool {
	o := defaultOptions()
	for _, option := range opts {
		option.apply(o)
	}
	pool := &Pool{
		maxWorkers:     o.maxWorkers,
		runningWorkers: make(map[int64]*worker),
		idleTimeout:    o.idleTimeout,
		workerSeq:      100,
		controlChan:    make(chan signal, 8),
		taskChan:       make(chan TaskFunc, o.maxTaskSize),
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

	waite := false
	stop := false

	for {
		if (stop || waite) && len(p.runningWorkers) == 0 {
			return
		}

		s := <-p.controlChan
		switch s.signal {
		case signalWorkerRoutinePanic:
			delete(p.runningWorkers, s.workerId)
			if !stop {
				p.startWorkerRoutine()
			}
		case signalWorkerRoutineExit:
			delete(p.runningWorkers, s.workerId)
		case signalWaiteWorkers:
			waite = true
			if s.idleTimeout > 0 {
				for _, w := range p.runningWorkers {
					w.wait <- s.idleTimeout
				}
			}
		case signalStopWorkers:
			stop = true
			for _, w := range p.runningWorkers {
				close(w.wait)
			}
		case signalAddTask:
			if !stop && atomic.LoadInt64(&p.idleWorkers) == 0 && len(p.runningWorkers) < p.maxWorkers {
				p.startWorkerRoutine()
			}
		}
	}
}

// start worker routine
func (p *Pool) startWorkerRoutine() {
	p.workerSeq++
	w := &worker{
		id:   p.workerSeq,
		wait: make(chan time.Duration, 1),
	}
	p.runningWorkers[p.workerSeq] = w

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				p.controlChan <- signal{signal: signalWorkerRoutinePanic, workerId: w.id}
				return
			}
			p.controlChan <- signal{signal: signalWorkerRoutineExit, workerId: w.id}
		}()

		idleTimeout := p.idleTimeout
		atomic.AddInt64(&p.idleWorkers, 1)

		for {
			select {
			case timeout, ok := <-w.wait:
				if !ok {
					return
				}
				idleTimeout = timeout
			case task := <-p.taskChan:
				atomic.AddInt64(&p.idleWorkers, -1)
				task()
				atomic.AddInt64(&p.idleWorkers, 1)
			case <-time.After(idleTimeout):
				return
			}
		}
	}()
}

// AddTask add a task to Pool, the worker routine will execute the task.
// if task channel is full, it will be blocked.
func (p *Pool) AddTask(task TaskFunc) {
	if task == nil {
		return
	}
	p.controlChan <- signal{signal: signalAddTask}
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
	p.controlChan <- signal{signal: signalStopWorkers}
	// waite
	p.wg.Wait()
}

// Wait blocking, waiting all tasks be executed and no tasks to execute in idle timeout(default 1s), then stop the pool,
// if taskChan never empty, it wouldn't return.
func (p *Pool) Wait() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// send waiting all routines done signal to control routine
	p.controlChan <- signal{signal: signalWaiteWorkers}
	// waite
	p.wg.Wait()
}

// WaitTimeout waiting with worker idle timeout, then stop the pool.
// if taskChan never empty, it wouldn't return.
func (p *Pool) WaitTimeout(idleTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// send waiting signal and idle timeout to control routine
	p.controlChan <- signal{signal: signalWaiteWorkers, idleTimeout: idleTimeout}
	// waite
	p.wg.Wait()
}
