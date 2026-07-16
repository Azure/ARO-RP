package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		New(),
		"exact",
		"embedded",
		"directuse",
		"shortstring",
		"github.com/Azure/ARO-RP/pkg/util/version",
	)
}
