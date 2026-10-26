package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePath(t *testing.T) {
	t.Setenv("CONFIG", "")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"among other flags", []string{"-a", "localhost:9090", "-r", "-c", "agent.json", "-l", "2"}, "agent.json"},
		{"long form", []string{"-config=server.json"}, "server.json"},
		{"two dashes", []string{"--c", "agent.json"}, "agent.json"},
		{"last one wins", []string{"-c", "first.json", "-c=second.json"}, "second.json"},
		{"no file", []string{"-a", "localhost:9090"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FilePath(tt.args))
		})
	}
}

func TestFilePathEnv(t *testing.T) {
	t.Setenv("CONFIG", "env.json")

	// The variable wins over the flag, like the other settings.
	assert.Equal(t, "env.json", FilePath([]string{"-c", "flag.json"}))
}

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"address": "localhost:9090", "poll_interval": "1m"}`), 0o600))

	cfg := struct {
		Address string `json:"address"`
	}{Address: "localhost:8080"}
	poll, report := int64(2), int64(10)

	err := LoadFile(path, &cfg, map[string]*int64{"poll_interval": &poll, "report_interval": &report})
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Address)
	assert.Equal(t, int64(60), poll)
	assert.Equal(t, int64(10), report, "a key missing from the file keeps its value")
}

func TestLoadFileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"poll_interval": "10"}`), 0o600))

	var poll int64
	err := LoadFile(path, &struct{}{}, map[string]*int64{"poll_interval": &poll})
	assert.Error(t, err, "an interval without a unit")
}
