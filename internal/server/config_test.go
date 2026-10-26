package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"restore": true,
		"store_interval": "1m",
		"store_file": "/tmp/from-file.json",
		"database_dsn": "postgres://file",
		"trusted_subnet": "192.168.1.0/24",
		"grpc_address": "localhost:3200"
	}`), 0o600))

	// The flag wins over the file, the variable wins over both.
	t.Setenv("CONFIG", "")
	t.Setenv("RESTORE", "false")

	cfg, err := Parse([]string{"-c", path, "-d", "postgres://flag"})
	require.NoError(t, err)

	assert.Equal(t, time.Minute, cfg.StoreInterval)
	assert.Equal(t, "/tmp/from-file.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://flag", cfg.DatabaseDSN)
	assert.False(t, cfg.Restore)
	assert.Equal(t, "192.168.1.0/24", cfg.TrustedNet.String())
	assert.Equal(t, "localhost:3200", cfg.GRPCAddr)
}

func TestParseTrustedSubnet(t *testing.T) {
	t.Setenv("CONFIG", "")

	_, err := Parse([]string{"-t", "192.168.1.0"})
	assert.Error(t, err, "a subnet without a mask")
}
