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
}

func (o *testObserver) Notify(event Event) error {
	o.events = append(o.events, event)
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

	// Close waits until the event is sent to all observers
	require.NoError(t, p.Close(context.Background()))

	assert.Equal(t, []Event{event}, first.events)
	assert.Equal(t, []Event{event}, second.events)
}
