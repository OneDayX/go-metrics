// Package audit sends audit events using the observer pattern.
package audit

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// queueSize is how many events an observer may fall behind before it starts
// losing new ones.
const queueSize = 100

// Event is one audit record about the metrics received by the server.
type Event struct {
	TS        int64    `json:"ts"`         // unix time in seconds
	Metrics   []string `json:"metrics"`    // names of the received metrics
	IPAddress string   `json:"ip_address"` // IP address of the client
}

// Observer receives audit events.
type Observer interface {
	// Notify handles one event.
	Notify(event Event) error
	// Close frees what the observer holds. Notify is not called after it.
	Close() error
}

// Publisher sends every event to all its observers. Each observer has its own
// queue and goroutine, so a slow observer holds up neither the request nor
// the other observers. It is safe for concurrent use.
type Publisher struct {
	log *zap.Logger

	mu     sync.Mutex
	queues map[Observer]chan Event // the queue of each observer

	wg sync.WaitGroup // observer goroutines that are still running
}

// NewPublisher returns a Publisher with no observers.
func NewPublisher(log *zap.Logger) *Publisher {
	return &Publisher{
		log:    log,
		queues: make(map[Observer]chan Event),
	}
}

// Subscribe adds an observer and starts its goroutine. Adding the same
// observer twice does nothing.
func (p *Publisher) Subscribe(o Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.queues[o]; ok {
		return
	}

	queue := make(chan Event, queueSize)
	p.queues[o] = queue
	p.wg.Go(func() { p.run(o, queue) })
}

// Unsubscribe removes an observer. It still gets the events already in its
// queue and is closed after them.
func (p *Publisher) Unsubscribe(o Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if queue, ok := p.queues[o]; ok {
		close(queue)
		delete(p.queues, o)
	}
}

// Publish puts the event in the queue of every observer. If the queue of an
// observer is full, that observer loses the event.
func (p *Publisher) Publish(event Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, queue := range p.queues {
		select {
		case queue <- event:
		default:
			p.log.Warn("audit queue is full, event is lost")
		}
	}
}

// Close removes all observers and waits until they send the events left in
// their queues and close.
func (p *Publisher) Close(ctx context.Context) error {
	p.mu.Lock()
	for _, queue := range p.queues {
		close(queue)
	}
	clear(p.queues)
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// run sends the events from the queue to the observer. When the queue is
// closed, it closes the observer, so Notify is never called on a closed one.
func (p *Publisher) run(o Observer, queue <-chan Event) {
	for event := range queue {
		if err := o.Notify(event); err != nil {
			p.log.Error("failed to send audit event", zap.Error(err))
		}
	}

	if err := o.Close(); err != nil {
		p.log.Error("failed to close audit observer", zap.Error(err))
	}
}
