package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

func TestAnalyzers(t *testing.T) {
	checks := analyzers()
	require.NoError(t, analysis.Validate(checks))

	names := make(map[string]bool)
	for _, a := range checks {
		names[a.Name] = true
	}

	// A typo in otherChecks would silently drop a check.
	for name := range otherChecks {
		assert.True(t, names[name], "check %s is not found", name)
	}
	assert.True(t, names["SA1000"])
	assert.True(t, names["osexit"])
}
