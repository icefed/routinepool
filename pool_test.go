package routinepool

import (
	"sync/atomic"
	"testing"
	"time"
)

type Count struct {
	Sum int32
}

func (c *Count) Do() {
	time.Sleep(time.Millisecond * 100)
	atomic.AddInt32(&c.Sum, 1)
}

func TestPoolStopAndStart(t *testing.T) {
	p := NewPool(0)
	p.Start()
	c := &Count{}
	for i := 0; i < 10; i++ {
		p.AddTask(c.Do)
	}
	go func() {
		for i := 0; i < 90; i++ {
			p.AddTask(c.Do)
		}
	}()
	p.Stop()
	p.Start()
	p.Waite()
	if c.Sum != 100 {
		t.Error("not finished")
	}
}

func TestPoolWaite(t *testing.T) {
	p := NewPool(10)
	p.Start()
	c := &Count{}
	for i := 0; i < 100; i++ {
		p.AddTask(c.Do)
	}
	p.Waite()
	if c.Sum != 100 {
		t.Error("not finished")
	}
}

type Err struct {
}

func (e *Err) Do() {
	panic("")
}

func TestPoolPanic(t *testing.T) {
	p := NewPool(10)
	p.Start()
	c := &Count{}
	for i := 0; i < 100; i++ {
		p.AddTask(c.Do)
		p.AddTask(func() {
			panic("")
		})
	}
	p.Waite()
	if c.Sum != 100 {
		t.Error("not finished")
	}
}
