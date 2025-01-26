package routinepool

import (
	"runtime"
)

type options struct {
	maxWorkers  int
	maxTaskSize int
}

func defaultOptions() *options {
	return &options{
		maxWorkers:  runtime.NumCPU(),
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

// WithMaxTaskSize set max task size, default use runtime.NumCPU().
func WithMaxTaskSize(maxTaskSize int) Option {
	return optionFunc{func(o *options) {
		if maxTaskSize <= 0 {
			maxTaskSize = runtime.NumCPU()
		}
		o.maxTaskSize = maxTaskSize
	}}
}
