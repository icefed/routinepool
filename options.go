package routinepool

import (
	"runtime"
	"time"
)

type options struct {
	maxWorkers  int
	idleTimeout time.Duration
	maxTaskSize int
}

func defaultOptions() *options {
	return &options{
		maxWorkers:  runtime.NumCPU(),
		idleTimeout: time.Second,
		maxTaskSize: runtime.NumCPU(),
	}
}

// Option defines pool options
type Option interface {
	apply(*options)
}

type optionFunc struct {
	f func(*options)
}

func (o optionFunc) apply(opts *options) {
	o.f(opts)
}

// WithMaxWorkers set max worker routines, default is runtime.NumCPU().
func WithMaxWorkers(maxWorkers int) Option {
	return optionFunc{func(o *options) {
		if maxWorkers <= 0 {
			maxWorkers = runtime.NumCPU()
		}
		o.maxWorkers = maxWorkers
	}}
}

// WithIdleTimeout set idle timeout, default is 1s. if idle timeout is less than 1s, it will be set to 1s.
// worker routine will be closed if no task received in idle timeout. if new task received, new worker will be created.
func WithIdleTimeout(idleTimeout time.Duration) Option {
	return optionFunc{func(o *options) {
		if idleTimeout < time.Second {
			idleTimeout = time.Second
		}
		o.idleTimeout = idleTimeout
	}}
}

// WithMaxTaskSize set max task size, default use runtime.NumCPU().
func WithMaxTaskSize(maxTaskSize int) Option {
	return optionFunc{func(o *options) {
		if maxTaskSize <= 0 {
			maxTaskSize = runtime.NumCPU()
		}
		o.maxTaskSize = maxTaskSize
	}}
}
