package pool_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/OneDayX/go-metrics/pkg/pool"
)

type buffer struct {
	data []byte
}

func (b *buffer) Reset() {
	b.data = b.data[:0]
}

func newBuffer() *buffer {
	return &buffer{data: make([]byte, 0, 64)}
}

func TestPutResets(t *testing.T) {
	p := pool.New(newBuffer)

	b := p.Get()
	b.data = append(b.data, "hello"...)
	p.Put(b)

	// b is checked after Put only to see that Reset was called.
	assert.Empty(t, b.data)
	assert.Equal(t, 64, cap(b.data))
}

func TestGetWithoutNew(t *testing.T) {
	p := pool.New[*buffer](nil)

	// Like sync.Pool without New, an empty pool returns the zero value.
	var b *buffer
	assert.NotPanics(t, func() { b = p.Get() })
	assert.Nil(t, b)
}

// Get a buffer, use it and put it back: the next Get returns a clean buffer.
func Example() {
	p := pool.New(newBuffer)

	b := p.Get()
	b.data = append(b.data, "hello"...)
	fmt.Println(string(b.data))
	p.Put(b)

	b = p.Get()
	fmt.Println(len(b.data))

	// Output:
	// hello
	// 0
}
