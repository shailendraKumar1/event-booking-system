package worker

import (
	"log"
	"sync"
	"sync/atomic"
)

// JobType distinguishes different kinds of background work.
type JobType string

const (
	JobBookingConfirmation JobType = "BOOKING_CONFIRMATION"
	JobEventNotification   JobType = "EVENT_NOTIFICATION"
)

// Job is a unit of background work dispatched to the pool.
type Job struct {
	Type    JobType
	Payload interface{}
}

// BookingPayload carries data for a booking confirmation email.
type BookingPayload struct {
	CustomerEmail string
	CustomerName  string
	EventTitle    string
	EventVenue    string
	EventDate     string
	Quantity      int
	BookingID     string
}

// EventUpdatePayload carries data for an event update notification.
type EventUpdatePayload struct {
	CustomerEmail string
	CustomerName  string
	EventTitle    string
	Changes       string
}

// Handler processes a single job. Implementations must be safe for concurrent use.
type Handler interface {
	Handle(job Job) error
}

// Pool is a bounded worker pool (Producer-Consumer pattern).
// Goroutine workers pull jobs from a shared channel, providing back-pressure
// via the channel buffer size.
type Pool struct {
	queue      chan Job
	handler    Handler
	wg         sync.WaitGroup
	quit       chan struct{}
	processed  atomic.Int64
	failed     atomic.Int64
}

func NewPool(poolSize, queueSize int, handler Handler) *Pool {
	p := &Pool{
		queue:   make(chan Job, queueSize),
		handler: handler,
		quit:    make(chan struct{}),
	}
	for i := 0; i < poolSize; i++ {
		p.wg.Add(1)
		go p.work(i)
	}
	log.Printf("[worker] pool started: %d workers, queue capacity %d", poolSize, queueSize)
	return p
}

func (p *Pool) work(id int) {
	defer p.wg.Done()
	for {
		select {
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			if err := p.handler.Handle(job); err != nil {
				p.failed.Add(1)
				log.Printf("[worker-%d] job %s failed: %v", id, job.Type, err)
			} else {
				p.processed.Add(1)
			}
		case <-p.quit:
			return
		}
	}
}

// Submit enqueues a job. Returns false if the queue is full (back-pressure).
func (p *Pool) Submit(job Job) bool {
	select {
	case p.queue <- job:
		return true
	default:
		log.Printf("[worker] queue full, dropping job type=%s", job.Type)
		return false
	}
}

// Stats returns processing metrics.
func (p *Pool) Stats() (processed, failed int64, queueLen int) {
	return p.processed.Load(), p.failed.Load(), len(p.queue)
}

// Shutdown gracefully drains the queue and stops workers.
func (p *Pool) Shutdown() {
	close(p.quit)
	close(p.queue)
	p.wg.Wait()
	log.Printf("[worker] pool shut down. processed=%d failed=%d", p.processed.Load(), p.failed.Load())
}
