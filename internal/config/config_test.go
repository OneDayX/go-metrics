package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
