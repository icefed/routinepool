package routinepool

import (
	"sync/atomic"
	"testing"
	"time"
)

type Count struct {
	sleep time.Duration
	Sum   int32
}

func (c *Count) Do() {
	time.Sleep(c.sleep)
	atomic.AddInt32(&c.Sum, 1)
}

func TestPoolStopAndStart(t *testing.T) {
	p := New()
	p.StartN(1)
	// already started
	p.Start()
	c := &Count{}
	go func() {
		for i := 0; i < 100; i++ {
			p.AddTask(c.Do)
		}
	}()
	p.Stop()
	p.StartN(100)
	p.Wait()
	if c.Sum != 100 {
		t.Error("not finished, want 100, got", c.Sum)
	}
}

func TestPoolWaite(t *testing.T) {
	p := New(WithMaxWorkers(10))
	p.Start()
	c := &Count{}
	for i := 0; i < 100; i++ {
		p.AddTask(c.Do)
	}
	p.Wait()
	if c.Sum != 100 {
		t.Error("not finished, want 100, got", c.Sum)
	}

	// should not block
	p.Stop()
	p.Wait()
}

func TestPoolWaiteTimeout(t *testing.T) {
	p := New(WithMaxWorkers(10))
	// not started
	p.WaitTimeout(time.Second * 3)

	p.Start()
	c := &Count{sleep: time.Millisecond * 100}
	for i := 0; i < 100; i++ {
		p.AddTask(c.Do)
	}

	now := time.Now()
	p.WaitTimeout(time.Second * 3)
	if c.Sum != 100 {
		t.Error("not finished, want 100, got", c.Sum)
	}
	duration := time.Since(now)
	if duration < time.Second*3 {
		t.Error("wait idle timeout not work")
	}
}

type Err struct {
}

func (e *Err) Do() {
	panic("")
}

func TestPoolPanic(t *testing.T) {
	p := New(WithMaxWorkers(10))
	p.Start()
	// nil task should be ignored
	p.AddTask(nil)
	c := &Count{}
	for i := 0; i < 100; i++ {
		p.AddTask(c.Do)
		p.AddTask(func() {
			panic("")
		})
	}
	p.Wait()
	if c.Sum != 100 {
		t.Error("not finished, want 100, got", c.Sum)
	}
}

func TestPoolPanicAfterStop(t *testing.T) {
	p := New()
	p.StartN(2)
	p.AddTask(func() {})
	p.AddTask(func() {
		time.Sleep(time.Second)
		panic("")
	})
	// p.Stop() should not block
	p.Stop()
}
