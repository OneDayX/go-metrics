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
	f, err := NewFileObserver(path)
	require.NoError(t, err)

	require.NoError(t, f.Notify(Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}))
	require.NoError(t, f.Notify(Event{TS: 2, Metrics: []string{"PollCount"}, IPAddress: "127.0.0.1"}))
	require.NoError(t, f.Close())

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	want := `{"ts":1,"metrics":["Alloc"],"ip_address":"127.0.0.1"}` + "\n" +
		`{"ts":2,"metrics":["PollCount"],"ip_address":"127.0.0.1"}` + "\n"
	assert.Equal(t, want, string(data))
}

func TestFileObserverBadPath(t *testing.T) {
	_, err := NewFileObserver(filepath.Join(t.TempDir(), "no-such-dir", "audit.log"))
	assert.Error(t, err)
}
