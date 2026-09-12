package retry

import (
	"errors"
	"testing"
	"time"
)

// TestDo checks how many attempts Do makes.
func TestDo(t *testing.T) {
	tests := []struct {
		name      string
		failures  int
		retriable bool
		wantCalls int
	}{
		{name: "succeeds at once", failures: 0, retriable: true, wantCalls: 1},
		{name: "succeeds on the second attempt", failures: 1, retriable: true, wantCalls: 2},
		{name: "gives up after three retries", failures: 99, retriable: true, wantCalls: 4},
		{name: "does not retry a permanent error", failures: 99, retriable: false, wantCalls: 1},
	}

	original := Delays
	Delays = []time.Duration{0, 0, 0}
	defer func() { Delays = original }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			err := Do(func() error {
				calls++
				if calls <= tt.failures {
					return errors.New("temporary")
				}
				return nil
			}, func(error) bool { return tt.retriable })

			if calls != tt.wantCalls {
				t.Errorf("made %d attempts, want %d", calls, tt.wantCalls)
			}
			if tt.failures == 0 && err != nil {
				t.Errorf("want success, got %v", err)
			}
		})
	}
}
