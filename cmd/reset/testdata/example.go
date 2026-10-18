package example

import "time"

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}

// Counter has its own Reset method.
type Counter struct {
	n int
}

func (c *Counter) Reset() {
	c.n = 0
}

func (c *Counter) Inc() {
	c.n++
}

// Kind is a named basic type.
type Kind string

type (
	// generate:reset
	Other struct {
		counter  Counter
		counterP *Counter
		kind     Kind
		ok       bool
		created  time.Time
		ids      [2]int
		err      error
		pp       **[]int
	}
)

// Ignored has no marker, Reset is not generated for it.
type Ignored struct {
	n int
}
