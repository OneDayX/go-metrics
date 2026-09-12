package sign

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	data := []byte("some data")
	hash := Sum(data, "key")

	tests := []struct {
		name string
		data []byte
		key  string
		hash string
		want bool
	}{
		{name: "valid hash", data: data, key: "key", hash: hash, want: true},
		{name: "wrong key", data: data, key: "other", hash: hash, want: false},
		{name: "wrong data", data: []byte("other data"), key: "key", hash: hash, want: false},
		{name: "not hex", data: data, key: "key", hash: "zzz", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Valid(tt.data, tt.key, tt.hash))
		})
	}
}
