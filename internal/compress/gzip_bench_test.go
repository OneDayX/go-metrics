package compress

import (
	"bytes"
	"testing"
)

func BenchmarkEncode(b *testing.B) {
	data := bytes.Repeat([]byte(`{"id":"Alloc","type":"gauge","value":2853240},`), 35)

	for b.Loop() {
		_, _ = Encode(data)
	}
}
