package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type job interface {
	Do(ctx context.Context) error
	Timeout() time.Duration
}

type pool struct {
	jobs chan job
	wg   sync.WaitGroup
}

func (p *pool) start(ctx context.Context, workerNum int) {
	if p.jobs != nil {
		panic("Cannot restart the same worker pool.")
	}
	if workerNum <= 0 {
		panic(fmt.Sprintf("A positive number for worker count is expected, but got %d", workerNum))
	}
	p.jobs = make(chan job)
	for range workerNum {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *pool) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return

		case j, ok := <-p.jobs:
			if !ok {
				return
			}
			process(ctx, j)
		}
	}
}

func process(ctx context.Context, j job) {
	t := j.Timeout()
	if t > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t)
		defer cancel()
	}
	err := j.Do(ctx)
	if err != nil {
		fmt.Println(err)
	}
}

// TODO(#142): Pass a context to set a timeout
func (p *pool) shutdown() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *pool) push(ctx context.Context, j job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case p.jobs <- j:
		return nil
	}
}
