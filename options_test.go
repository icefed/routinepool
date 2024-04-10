package routinepool

import (
	"runtime"
	"testing"
	"time"
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

	WithIdleTimeout(time.Second * 2).apply(o)
	if o.idleTimeout != time.Second*2 {
		t.Errorf("WithIdleTimeout want %d, got %d", time.Second*2, o.idleTimeout)
	}
	WithIdleTimeout(time.Millisecond * 100).apply(o)
	if o.idleTimeout != time.Second {
		t.Errorf("WithIdleTimeout want %d, got %d", time.Second, o.idleTimeout)
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
