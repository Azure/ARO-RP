package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
)

const (
	analyzerName      = "constdup"
	sourcePackagePath = "github.com/Azure/ARO-RP/pkg/util/version"
)

var registryValuePattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:[-+._][A-Za-z0-9._-]+)*$`)

type sharedConst struct {
	Name  string
	Value string
}

type sharedConstCache struct {
	once      sync.Once
	sourceDir string
	values    []sharedConst
	err       error
}

func New() *analysis.Analyzer {
	cache := &sharedConstCache{}

	return &analysis.Analyzer{
		Name: analyzerName,
		Doc:  "report raw string literals that duplicate registry-worthy constants from pkg/util/version",
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, cache)
		},
	}
}

func run(pass *analysis.Pass, cache *sharedConstCache) (any, error) {
	if len(pass.Files) == 0 {
		return nil, nil
	}

	if strings.HasPrefix(pass.Pkg.Path(), sourcePackagePath) {
		return nil, nil
	}

	values, sourceDir, err := cache.load(pass)
	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	for _, file := range pass.Files {
		filename := pass.Fset.PositionFor(file.Package, false).Filename
		if shouldSkipAnalyzedFile(filename, sourceDir, file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}

			match, exact := matchSharedConst(value, values)
			if match == nil {
				return true
			}

			if exact {
				pass.Reportf(lit.Pos(), "string literal duplicates shared constant version.%s; use the shared constant instead", match.Name)
			} else {
				pass.Reportf(lit.Pos(), "string literal embeds shared constant version.%s; prefer composing from the shared constant", match.Name)
			}

			return true
		})
	}

	return nil, nil
}

func (c *sharedConstCache) load(pass *analysis.Pass) ([]sharedConst, string, error) {
	c.once.Do(func() {
		c.sourceDir, c.values, c.err = loadSharedConsts(pass)
	})

	return c.values, c.sourceDir, c.err
}

func loadSharedConsts(pass *analysis.Pass) (string, []sharedConst, error) {
	filename := firstFilename(pass)
	if filename == "" {
		return "", nil, nil
	}

	sourceDir, err := resolveSourceDir(filename)
	if err != nil {
		return "", nil, err
	}

	values, err := collectSharedConsts(sourceDir)
	if err != nil {
		return "", nil, err
	}

	return sourceDir, values, nil
}

func firstFilename(pass *analysis.Pass) string {
	for _, file := range pass.Files {
		if file == nil {
			continue
		}

		filename := pass.Fset.PositionFor(file.Package, false).Filename
		if filename != "" {
			return filename
		}
	}

	return ""
}

func resolveSourceDir(filename string) (string, error) {
	dir := filepath.Dir(filename)

	for {
		if filepath.Base(dir) == "src" {
			sourceDir := filepath.Join(dir, filepath.FromSlash(sourcePackagePath))
			if directoryExists(sourceDir) {
				return sourceDir, nil
			}
		}

		modulePath, err := readModulePath(filepath.Join(dir, "go.mod"))
		if err == nil {
			if rel, ok := strings.CutPrefix(sourcePackagePath, modulePath+"/"); ok {
				sourceDir := filepath.Join(dir, filepath.FromSlash(rel))
				if directoryExists(sourceDir) {
					return sourceDir, nil
				}
			}
			if modulePath == sourcePackagePath && directoryExists(dir) {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("constdup: unable to locate %s from %s", sourcePackagePath, filename)
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func readModulePath(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("constdup: module directive not found in %s", goModPath)
}

func collectSharedConsts(sourceDir string) ([]sharedConst, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("constdup: read source package: %w", err)
	}

	fset := token.NewFileSet()
	values := make([]sharedConst, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filename := filepath.Join(sourceDir, name)
		file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("constdup: parse source package file %s: %w", filename, err)
		}

		if isGeneratedFile(file) {
			continue
		}

		values = append(values, collectConstsFromFile(file)...)
	}

	sort.Slice(values, func(i, j int) bool {
		return len(values[i].Value) > len(values[j].Value)
	})

	return values, nil
}

func collectConstsFromFile(file *ast.File) []sharedConst {
	values := make([]sharedConst, 0)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}

				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}

				value, err := strconv.Unquote(lit.Value)
				if err != nil || !isRegistryWorthyValue(value) {
					continue
				}

				values = append(values, sharedConst{
					Name:  name.Name,
					Value: value,
				})
			}
		}
	}

	return values
}

func isRegistryWorthyValue(value string) bool {
	return strings.Contains(value, "@sha256:") ||
		(strings.Contains(value, "/") && strings.Contains(value, ":")) ||
		registryValuePattern.MatchString(value)
}

func shouldSkipAnalyzedFile(filename, sourceDir string, file *ast.File) bool {
	if file == nil || isGeneratedFile(file) {
		return true
	}

	normalizedFile := normalizePath(filename)
	normalizedSourceDir := normalizePath(sourceDir)

	if normalizedSourceDir != "" && (normalizedFile == normalizedSourceDir || strings.HasPrefix(normalizedFile, normalizedSourceDir+"/")) {
		return true
	}

	return isExcludedPath(normalizedFile)
}

func isGeneratedFile(file *ast.File) bool {
	return ast.IsGenerated(file)
}

func isExcludedPath(filename string) bool {
	if filename == "" {
		return false
	}

	base := filepath.Base(filename)
	if strings.HasPrefix(base, "zz_generated_") || base == "bindata.go" {
		return true
	}

	normalized := "/" + strings.TrimPrefix(filename, "/")
	for _, pathPart := range []string{
		"/vendor/",
		"/third_party/",
		"/builtin/",
		"/examples/",
		"/pkg/client/",
	} {
		if strings.Contains(normalized, pathPart) {
			return true
		}
	}

	return false
}

func normalizePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func matchSharedConst(literal string, values []sharedConst) (*sharedConst, bool) {
	for i := range values {
		value := &values[i]

		if literal == value.Value {
			return value, true
		}

		if strings.Contains(literal, value.Value) {
			return value, false
		}
	}

	return nil, false
}
