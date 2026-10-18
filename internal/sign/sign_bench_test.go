package sign

import (
	"bytes"
	"testing"
)

func BenchmarkSum(b *testing.B) {
	data := bytes.Repeat([]byte(`{"id":"Alloc","type":"gauge","value":2853240},`), 35)

	for b.Loop() {
		Sum(data, "secret")
	}
}
