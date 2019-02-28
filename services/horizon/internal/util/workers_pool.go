package util

import (
	"sync"
)

// WorkersPool provides a simple synchronization between workers doing some work.
// First add jobs by using AddWork(), then add a worker function using
// `SetWorker`. Finally, start as many workers as you want using `Start`.
type WorkersPool struct {
	work   []interface{}
	worker func(workerID int, work interface{})
	mutex  sync.Mutex
}

func (w *WorkersPool) AddWork(job interface{}) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.work == nil {
		w.work = make([]interface{}, 0)
	}

	w.work = append(w.work, job)
}

func (w *WorkersPool) WorkSize() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return len(w.work)
}

// getWork returns a work. The second return value is false if there's no more work.
func (w *WorkersPool) getWork() (interface{}, bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(w.work) == 0 {
		return nil, false
	}

	var work interface{}
	work, w.work = w.work[len(w.work)-1], w.work[:len(w.work)-1]
	return work, true
}

func (w *WorkersPool) SetWorker(worker func(int, interface{})) {
	w.worker = worker
}

// Start starts working using `workers` workers
func (w *WorkersPool) Start(workers int) {
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			for {
				job, ok := w.getWork()
				if !ok {
					wg.Done()
					return
				}

				w.worker(workerID, job)
			}
		}(i)
	}
	wg.Wait()
}
