package constdup

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/Azure/ARO-RP/tools/constdup/analyzer"
)

func init() {
	register.Plugin("constdup", New)
}

type plugin struct{}

func New(any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.New()}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
