package compress

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	const data = `{"id":"Alloc","type":"gauge","value":123.45}`

	encoded, err := Encode([]byte(data))
	require.NoError(t, err)
	assert.NotEqual(t, data, string(encoded))

	decoded, err := Decode(bytes.NewReader(encoded))
	require.NoError(t, err)
	assert.Equal(t, data, string(decoded))
}
