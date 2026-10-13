// Package audit sends audit events using the observer pattern.
package audit

import (
	"context"

	"go.uber.org/zap"
)

// Event is one audit record about the metrics received by the server.
type Event struct {
	TS        int64    `json:"ts"`         // unix time in seconds
	Metrics   []string `json:"metrics"`    // names of the received metrics
	IPAddress string   `json:"ip_address"` // IP address of the client
}

// Observer receives audit events.
type Observer interface {
	Notify(event Event) error
}

// Publisher sends every event to all its observers. Events are sent in a
// separate goroutine, so the request does not wait for them.
type Publisher struct {
	log       *zap.Logger
	observers []Observer
	events    chan Event
	done      chan struct{}
}

// NewPublisher returns a Publisher with no observers and starts its sending
// goroutine. Call Close to stop it.
func NewPublisher(log *zap.Logger) *Publisher {
	p := &Publisher{
		log:    log,
		events: make(chan Event, 100),
		done:   make(chan struct{}),
	}
	go p.run()
	return p
}

// Subscribe adds an observer. Call it before the server starts.
func (p *Publisher) Subscribe(o Observer) {
	p.observers = append(p.observers, o)
}

// Publish puts the event in the queue. If the queue is full, the event is lost.
func (p *Publisher) Publish(event Event) {
	select {
	case p.events <- event:
	default:
		p.log.Warn("audit queue is full, event is lost")
	}
}

// Close waits until all events from the queue are sent.
func (p *Publisher) Close(ctx context.Context) error {
	close(p.events)

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Publisher) run() {
	defer close(p.done)

	for event := range p.events {
		for _, o := range p.observers {
			if err := o.Notify(event); err != nil {
				p.log.Error("failed to send audit event", zap.Error(err))
			}
		}
	}
}
