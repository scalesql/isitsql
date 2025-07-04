package app

import (
	"runtime"
	"sync"
)

type Pool struct {
	sync.Mutex
	size  int
	tasks chan string
	kill  chan struct{}
	wg    sync.WaitGroup
}

// NewPool is...
func NewPool(size int) *Pool {
	pool := &Pool{
		tasks: make(chan string),
		kill:  make(chan struct{}),
	}
	pool.Resize(size)
	return pool
}

func (p *Pool) worker() {
	// defer p.wg.Done()
	// for {
	// 	select {
	// 	case task, ok := <-p.tasks:
	// 		if !ok {
	// 			return
	// 		}
	// 		pollServer(task)
	// 	case <-p.kill:
	// 		return
	// 	}
	// }
}

// Resize the pool
func (p *Pool) Resize(n int) {

	if n == 0 {
		n = runtime.NumCPU() * 4
		if n < 8 {
			n = 8
		}
	}

	p.Lock()
	defer p.Unlock()
	for p.size < n {
		p.size++
		p.wg.Add(1)
		go p.worker()
	}
	for p.size > n {
		p.size--
		p.kill <- struct{}{}
	}
}

// Close the channel
func (p *Pool) Close() {
	close(p.tasks)
}

// Wait for all to close
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Poll initiates a polling
func (p *Pool) Poll(task string) {
	p.tasks <- task
}
