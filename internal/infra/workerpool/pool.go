package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sales-system/go-real-time/internal/infra/logger"
)

// Task represents an executable background job.
type Task func(ctx context.Context) error

// WorkerPool manages a concurrent pool of goroutines for processing non-blocking background tasks.
type WorkerPool struct {
	workers    int
	queue      chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownMu sync.Mutex
	isClosed   bool
}

// New creates and starts a bounded worker pool.
func New(numWorkers, queueCapacity int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:  numWorkers,
		queue:    make(chan Task, queueCapacity),
		ctx:      ctx,
		cancel:   cancel,
		isClosed: false,
	}

	for i := 0; i < numWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	logger.Info("Background worker pool started", logger.Fields{
		"workers":  numWorkers,
		"capacity": queueCapacity,
	})

	return pool
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Drain remaining tasks if any before exiting
			for {
				select {
				case task, ok := <-p.queue:
					if !ok {
						return
					}
					p.safeExecute(id, task)
				default:
					return
				}
			}
		case task, ok := <-p.queue:
			if !ok {
				return
			}
			p.safeExecute(id, task)
		}
	}
}

func (p *WorkerPool) safeExecute(workerID int, task Task) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic recovered in worker execution", logger.Fields{
				"worker_id": workerID,
				"panic":     fmt.Sprintf("%v", r),
			})
		}
	}()

	taskCtx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	if err := task(taskCtx); err != nil {
		logger.Error("Background task execution failed", logger.Fields{
			"worker_id": workerID,
			"error":     err.Error(),
		})
	}
}

// Submit queues a task for background asynchronous execution.
func (p *WorkerPool) Submit(task Task) bool {
	p.shutdownMu.Lock()
	defer p.shutdownMu.Unlock()

	if p.isClosed {
		logger.Warn("Failed to submit task: worker pool is shutting down")
		return false
	}

	select {
	case p.queue <- task:
		return true
	default:
		logger.Warn("Worker pool queue is full; task dropped or handling under pressure")
		return false
	}
}

// Shutdown gracefully waits for current tasks to complete.
func (p *WorkerPool) Shutdown(timeout time.Duration) {
	p.shutdownMu.Lock()
	if p.isClosed {
		p.shutdownMu.Unlock()
		return
	}
	p.isClosed = true
	p.shutdownMu.Unlock()

	close(p.queue)
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Worker pool stopped cleanly")
	case <-time.After(timeout):
		logger.Warn("Worker pool shutdown timed out; forced exit")
	}
}
