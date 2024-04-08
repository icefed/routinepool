package pool

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

	signalChan chan int
	taskChan   chan TaskFunc
	stopChan   chan struct{}
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
func NewPool(maxRoutines int) *Pool {
	if maxRoutines <= 0 {
		maxRoutines = runtime.NumCPU()
	}
	pool := &Pool{
		maxRoutines: maxRoutines,
		signalChan:  make(chan int, 8),
		stopChan:    make(chan struct{}, maxRoutines),
		taskChan:    make(chan TaskFunc, maxRoutines),
	}
	return pool
}

// Start start pool routines in background
func (p *Pool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.runningRoutines = 1
	p.wg.Add(1)
	go p.startWorkerRoutine(time.Second)

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

		signal := <-p.signalChan
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
				p.stopChan <- struct{}{}
			}
		case signalNoTaskTimeout:
			if p.runningRoutines > 1 || waite {
				p.stopChan <- struct{}{}
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
			p.signalChan <- signalWorkerRoutinePanic
			return
		}
		p.signalChan <- signalWorkerRoutineExit
	}()
	for {
		select {
		case <-p.stopChan:
			return
		case task := <-p.taskChan:
			if task != nil {
				task()
			}
		case <-time.After(noTaskTimeout): // if there is no task after noTaskTimeout, it will send no task signal to controller
			p.signalChan <- signalNoTaskTimeout
		}
	}
}

// AddTask add a task to Pool, the worker routine will execute the task.
// if task channel is full, it will block.
func (p *Pool) AddTask(task TaskFunc) {
	p.signalChan <- signalAddTask
	p.taskChan <- task
}

// Stop stop all routines no matter how many tasks wait for execution.
func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// send stop all routines signal to control routine
	p.signalChan <- signalStopRoutines
	// waite
	p.wg.Wait()
}

// Waite blocking, waiting all tasks be executed and durtion time no tasks to execute, then stop the pool,
// if taskChan never empty, it wouldn't return.
func (p *Pool) Waite() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// send waiting all routines done signal to control routine
	p.signalChan <- signalWaiteRoutines
	// waite
	p.wg.Wait()
}
