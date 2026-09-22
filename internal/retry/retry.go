// Package retry repeats an operation that failed with a temporary error.
package retry

import "time"

// Delays is the retry policy: one attempt, then one more after each delay.
// A variable so that tests can shorten the waits.
var Delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func Do(fn func() error, isRetriable func(error) bool) error {
	err := fn()

	for _, delay := range Delays {
		if err == nil || !isRetriable(err) {
			return err
		}

		time.Sleep(delay)
		err = fn()
	}

	return err
}
