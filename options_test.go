package routinepool

import (
	"runtime"
	"testing"
)

func TestOptions(t *testing.T) {
	o := defaultOptions()
	WithMaxWorkers(10).apply(o)
	if o.maxWorkers != 10 {
		t.Errorf("WithMaxWorkers want 10, got %d", o.maxWorkers)
	}
	WithMaxWorkers(0).apply(o)
	if o.maxWorkers != runtime.NumCPU() {
		t.Errorf("WithMaxWorkers want %d, got %d", runtime.NumCPU(), o.maxWorkers)
	}

	WithMaxTaskSize(10).apply(o)
	if o.maxTaskSize != 10 {
		t.Errorf("WithMaxTaskSize want 10, got %d", o.maxTaskSize)
	}
	WithMaxTaskSize(0).apply(o)
	if o.maxTaskSize != runtime.NumCPU() {
		t.Errorf("WithMaxTaskSize want %d, got %d", runtime.NumCPU(), o.maxTaskSize)
	}
}
