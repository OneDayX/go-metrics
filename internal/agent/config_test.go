package agent

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig saves a config file and returns its path.
func writeConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	return path
}

// setArgs replaces the command line and the flag set that GetConfig uses.
func setArgs(t *testing.T, args ...string) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })

	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ExitOnError)
}

func TestGetConfig(t *testing.T) {
	path := writeConfig(t, `{
		"poll_interval": "5s",
		"report_interval": "20s",
		"rate_limit": 3,
		"key": "file-key"
	}`)

	// The flag wins over the file, the variable wins over both.
	setArgs(t, "agent", "-r", "30")
	t.Setenv("CONFIG", path)
	t.Setenv("KEY", "env-key")

	cfg := GetConfig()

	assert.Equal(t, 5*time.Second, cfg.PollInterval)
	assert.Equal(t, 30*time.Second, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.RateLimit)
	assert.Equal(t, "env-key", cfg.Key)
}

func TestLoadFileError(t *testing.T) {
	path := writeConfig(t, `{"poll_interval": "often"}`)
	var poll, report int64

	err := loadFile(path, &Config{}, &poll, &report)
	assert.Error(t, err)
}
