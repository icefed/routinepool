package routinepool

import (
	"runtime"
	"sync"
	"time"
)

// Pool routine pool
type Pool struct {
	mu sync.Mutex
	wg sync.WaitGroup

	maxRoutines     int // max routine numbers
	runningRoutines int // in running routine numbers

	controlChan    chan int
	taskChan       chan TaskFunc
	stopWorkerChan chan struct{}
}

// TaskFunc defines a task function
type TaskFunc func()

const (
	signalWorkerRoutineExit = iota
	signalWorkerRoutinePanic
	signalWaiteRoutines
	signalStopRoutines
	signalNoTaskTimeout
	signalAddTask
)

// NewPool create a routine pool, param maxRoutines set the max routine numbers can be used in pool.
// if maxRoutines <= 0, it will be set to runtime.NumCPU().
// task channel size will be set to maxRoutines.
func NewPool(maxRoutines int) *Pool {
	if maxRoutines <= 0 {
		maxRoutines = runtime.NumCPU()
	}
	pool := &Pool{
		maxRoutines:    maxRoutines,
		controlChan:    make(chan int, 8),
		stopWorkerChan: make(chan struct{}, maxRoutines),
		taskChan:       make(chan TaskFunc, maxRoutines),
	}
	return pool
}

// Start start routine pool in background.
// The initial number of workers is determined by the number of tasks in the channel.
func (p *Pool) Start() {
	p.start(min(len(p.taskChan), p.maxRoutines))
}

// StartN start routine pool in background with workerNum initial workers.
// if workerNum <= 0 or workerNum > maxRoutines, it will be set to maxRoutines.
func (p *Pool) StartN(workerNum int) {
	if workerNum < 0 || workerNum > p.maxRoutines {
		workerNum = p.maxRoutines
	}
	p.start(workerNum)
}

func (p *Pool) start(workerNum int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.runningRoutines = workerNum
	p.wg.Add(workerNum)
	for i := 0; i < workerNum; i++ {
		go p.startWorkerRoutine(time.Second)
	}

	p.wg.Add(1)
	go p.startControllerRoutine()
}

// start controller routine
func (p *Pool) startControllerRoutine() {
	defer p.wg.Done()

	waite := false
	stop := false

	for {
		if (stop || waite) && p.runningRoutines == 0 {
			return
		}

		signal := <-p.controlChan
		switch signal {
		case signalWorkerRoutinePanic:
			if !stop {
				p.wg.Add(1)
				go p.startWorkerRoutine(time.Second)
			}
		case signalWorkerRoutineExit:
			p.runningRoutines--
		case signalWaiteRoutines:
			waite = true
		case signalStopRoutines:
			stop = true
			for i := 0; i < p.runningRoutines; i++ {
				p.stopWorkerChan <- struct{}{}
			}
		case signalNoTaskTimeout:
			if p.runningRoutines > 1 || waite {
				p.stopWorkerChan <- struct{}{}
			}
		case signalAddTask:
			if !stop && p.runningRoutines < p.maxRoutines {
				p.runningRoutines++
				p.wg.Add(1)
				go p.startWorkerRoutine(time.Second)
			}
		}
	}
}

// start worker routine
func (p *Pool) startWorkerRoutine(noTaskTimeout time.Duration) {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.controlChan <- signalWorkerRoutinePanic
			return
		}
		p.controlChan <- signalWorkerRoutineExit
	}()
	for {
		select {
		case <-p.stopWorkerChan:
			return
		case task := <-p.taskChan:
			task()
		case <-time.After(noTaskTimeout): // if there is no task after noTaskTimeout, it will send no task signal to controller
			p.controlChan <- signalNoTaskTimeout
		}
	}
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

	// send stop all routines signal to control routine
	p.controlChan <- signalStopRoutines
	// waite
	p.wg.Wait()
}

// Waite blocking, waiting all tasks be executed and durtion time no tasks to execute, then stop the pool,
// if taskChan never empty, it wouldn't return.
func (p *Pool) Waite() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// send waiting all routines done signal to control routine
	p.controlChan <- signalWaiteRoutines
	// waite
	p.wg.Wait()
}
