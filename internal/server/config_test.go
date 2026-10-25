package server

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
	path := filepath.Join(t.TempDir(), "server.json")
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
		"restore": true,
		"store_interval": "1m",
		"store_file": "/tmp/from-file.json",
		"database_dsn": "postgres://file",
		"trusted_subnet": "192.168.1.0/24",
		"grpc_address": "localhost:3200"
	}`)

	// The flag wins over the file, the variable wins over both.
	setArgs(t, "server", "-c", path, "-d", "postgres://flag")
	t.Setenv("RESTORE", "false")

	cfg := GetConfig()

	assert.Equal(t, time.Minute, cfg.StoreInterval)
	assert.Equal(t, "/tmp/from-file.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://flag", cfg.DatabaseDSN)
	assert.False(t, cfg.Restore)
	assert.Equal(t, "192.168.1.0/24", cfg.TrustedNet.String())
	assert.Equal(t, "localhost:3200", cfg.GRPCAddr)
}

func TestLoadFileError(t *testing.T) {
	path := writeConfig(t, `{"store_interval": "10"}`)
	seconds := defaultStoreIntervalSeconds

	err := loadFile(path, &Config{}, &seconds)
	assert.Error(t, err, "an interval without a unit")
}
