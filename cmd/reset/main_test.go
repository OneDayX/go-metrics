package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	// testdata is skipped by ./..., so the example is copied to a new module.
	dir := t.TempDir()
	src, err := os.ReadFile("testdata/example.go")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "example.go"), src, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o644))

	require.NoError(t, run(dir))

	got, err := os.ReadFile(filepath.Join(dir, genFile))
	require.NoError(t, err)
	want, err := os.ReadFile("testdata/reset.gen.go.golden")
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

func TestRunNoPackages(t *testing.T) {
	assert.Error(t, run(t.TempDir()))
}
