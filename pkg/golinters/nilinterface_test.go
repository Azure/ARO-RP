package golinters

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdatadir := filepath.Join(filepath.Dir(filepath.Dir(wd)), "test/lint/")
	analysistest.Run(t, testdatadir, NilInterfaceAnalyser, "nilinterface")
}
