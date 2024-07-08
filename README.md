# routine pool - Golang routine pool for multi asynchronous tasks.

[![GitHub Tag](https://img.shields.io/github/v/tag/icefed/routinepool)](https://github.com/icefed/routinepool/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/icefed/routinepool.svg)](https://pkg.go.dev/github.com/icefed/routinepool)
[![Go Report Card](https://goreportcard.com/badge/github.com/icefed/routinepool)](https://goreportcard.com/report/github.com/icefed/routinepool)
![Build Status](https://github.com/icefed/routinepool/actions/workflows/test.yml/badge.svg)
[![Coverage](https://img.shields.io/codecov/c/github/icefed/routinepool)](https://codecov.io/gh/icefed/routinepool)
[![License](https://img.shields.io/github/license/icefed/routinepool)](./LICENSE)

### Supports
- Automatic scaling Number of routines by the number of tasks.
- Support waiting all tasks finished.
- Panic in tasks will be recovered.

### Usages

Write a http benchmark tool with routinepool, calculate the average time of each request.
```
package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icefed/routinepool"
)

func main() {
	p := routinepool.New(routinepool.WithMaxWorkers(8))
	p.StartN(8)

	var errCount int32
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 8,
		},
	}
	costs := make([]time.Duration, 0)
	mu := sync.Mutex{}

	f := func() {
		start := time.Now()
		defer func() {
			mu.Lock()
			defer mu.Unlock()
			costs = append(costs, time.Since(start))
		}()
		req, _ := http.NewRequest("GET", "http://localhost:8099/hello", nil)
		resp, err := client.Do(req)
		io.Copy(io.Discard, resp.Body)
		if err != nil {
			atomic.AddInt32(&errCount, 1)
			return
		}
		resp.Body.Close()
	}
	for i := 0; i < 100000; i++ {
		p.AddTask(f)
	}
	p.Wait()

	avg := time.Duration(0)
	total := time.Duration(0)
	for _, cost := range costs {
		total += cost
	}
	avg = total / time.Duration(len(costs))
	fmt.Printf("total requests: %d, avg cost: %s, err count: %d\n", len(costs), avg, errCount)
}
```
