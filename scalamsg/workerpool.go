package scalamsg

import (
	"sync"
	"sync/atomic"
)

// WorkerPool is a pool of workers.
type WorkerPool struct {
	wg sync.WaitGroup
	sync.Mutex
	workers []*chan struct{}
	cnt     int32
}

// NewWorkerPool creates a new WorkerPool.
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{}
}

// Worker is a function with a cancel channel it should check for exit.
type Worker func(done <-chan struct{})

// AddWorker adds a worker into the WorkerPool.
func (wp *WorkerPool) AddWorker(w Worker) {
	done := make(chan struct{})
	wp.Lock()
	defer wp.Unlock()
	added := false
	for i, pc := range wp.workers {
		if *pc == nil {
			wp.workers[i] = &done
			added = true
		}
	}
	if !added {
		wp.workers = append(wp.workers, &done)
	}
	atomic.AddInt32(&wp.cnt, 1)
	wp.wg.Add(1)
	go func() {
		defer func() {
			done = nil
			atomic.AddInt32(&wp.cnt, -1)
			wp.wg.Done()
		}()
		w(done)
	}()
}

// RmWorker tries to remove a random running worker from the WorkerPool.
// It may do nothing if all workers in the WorkerPool have been removed
// or exited.
func (wp *WorkerPool) RmWorker() {
	wp.Lock()
	defer wp.Unlock()
	var pdone *chan struct{}
	for _, pc := range wp.workers {
		if *pc != nil {
			pdone = pc
		}
	}
	if pdone == nil {
		return
	}
	close(*pdone)
	*pdone = nil
}

// Close closes and waits all the workers in the WorkerPool to exit.
func (wp *WorkerPool) Close() {
	wp.Lock()
	defer wp.Unlock()
	for _, pc := range wp.workers {
		if *pc != nil {
			close(*pc)
		}
	}
	wp.wg.Wait()
}

// Len returns the number of running workers in the WorkerPool.
func (wp *WorkerPool) Len() int {
	cnt := atomic.LoadInt32(&wp.cnt)
	return int(cnt)
}
