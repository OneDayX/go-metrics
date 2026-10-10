package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileObserver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	f := NewFileObserver(path)

	require.NoError(t, f.Notify(Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}))
	require.NoError(t, f.Notify(Event{TS: 2, Metrics: []string{"PollCount"}, IPAddress: "127.0.0.1"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	want := `{"ts":1,"metrics":["Alloc"],"ip_address":"127.0.0.1"}` + "\n" +
		`{"ts":2,"metrics":["PollCount"],"ip_address":"127.0.0.1"}` + "\n"
	assert.Equal(t, want, string(data))
}
