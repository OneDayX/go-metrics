// Package retry repeats an operation that failed with a temporary error.
package retry

import "time"

var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func Do(fn func() error, isRetriable func(error) bool) error {
	err := fn()

	for _, delay := range delays {
		if err == nil || !isRetriable(err) {
			return err
		}

		time.Sleep(delay)
		err = fn()
	}

	return err
}
