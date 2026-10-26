package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"poll_interval": "5s",
		"report_interval": "20s",
		"rate_limit": 3,
		"key": "file-key",
		"grpc_address": "localhost:3200"
	}`), 0o600))

	// The flag wins over the file, the variable wins over both.
	t.Setenv("CONFIG", path)
	t.Setenv("KEY", "env-key")

	cfg, err := Parse([]string{"-r", "30"})
	require.NoError(t, err)

	assert.Equal(t, 5*time.Second, cfg.PollInterval)
	assert.Equal(t, 30*time.Second, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.RateLimit)
	assert.Equal(t, "env-key", cfg.Key)
	assert.Equal(t, "localhost:3200", cfg.GRPCAddr)
}

func TestParseRateLimit(t *testing.T) {
	t.Setenv("CONFIG", "")

	_, err := Parse([]string{"-l", "0"})
	assert.Error(t, err, "no workers would read the batches")
}
