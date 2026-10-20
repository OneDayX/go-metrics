package osexit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/OneDayX/go-metrics/cmd/staticlint/osexit"
)

func TestAnalyzer(t *testing.T) {
	// a is a main package with os.Exit in main, b is not a main package.
	analysistest.Run(t, analysistest.TestData(), osexit.Analyzer, "a", "b")
}
