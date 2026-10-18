package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testObserver struct {
	events []Event
	closed bool
}

func (o *testObserver) Notify(event Event) error {
	o.events = append(o.events, event)
	return nil
}

func (o *testObserver) Close() error {
	o.closed = true
	return nil
}

func TestPublisher(t *testing.T) {
	first := &testObserver{}
	second := &testObserver{}

	p := NewPublisher(zap.NewNop())
	p.Subscribe(first)
	p.Subscribe(second)

	event := Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	p.Publish(event)

	// Close waits until the event is sent to all observers and closes them
	require.NoError(t, p.Close(context.Background()))

	assert.Equal(t, []Event{event}, first.events)
	assert.Equal(t, []Event{event}, second.events)
	assert.True(t, first.closed)
	assert.True(t, second.closed)
}

func TestPublisherUnsubscribe(t *testing.T) {
	kept := &testObserver{}
	removed := &testObserver{}

	p := NewPublisher(zap.NewNop())
	p.Subscribe(kept)
	p.Subscribe(removed)
	p.Unsubscribe(removed)

	p.Publish(Event{TS: 1, Metrics: []string{"Alloc"}})
	require.NoError(t, p.Close(context.Background()))

	assert.Len(t, kept.events, 1)
	assert.Empty(t, removed.events)
	assert.True(t, removed.closed)
}
