/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/gopackages"
	"github.com/dave/dst/dstutil"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

// The structure of api.NewCloudError leads to a couple types of errors that are subtle and hard to track:
// 1. Non-constant format strings, e.g. some dynamic string being used as the first argument to fmt.Sprintf().
//    When that string happens to contain %s or similar, the output will be mangled.
// 2. Lack of the normal linting rules for argument count or type. Normally, mistakes will cause lint failures:
//    printf: fmt.Sprintf call needs 2 args but has 3 args (govet)
//    printf: fmt.Sprintf format %d has arg cloud of wrong type string (govet)
//
// This file uses the Go AST to dynamically refactor every call to api.NewCloudError to bring the formatting up
// a level - so, the new contract for api.NewCloudError goes ...
// from: func NewCloudError(statusCode int, code, target, message string, a ...interface{}) *CloudError
// to:   func NewCloudError(statusCode int, code, target, message string) *CloudError
//
// In order to achieve this, we make the following refactors:
// 1. Calls that never provided format arguments are kept as-is:
//    api.NewCloudError(code,reason,target,message) -> api.NewCloudError(code,reason,target,message)
// 2. Calls that provided a dynamic, pre-formatted string are kept as-is:
//    api.NewCloudError(code,reason,target,fmt.Sprintf(fmt,args...)) -> api.NewCloudError(code,reason,target,fmt.Sprintf(fmt,args...))
// 3. Calls that provide arguments to a constant format string are wrapped in fmt.Sprintf()
//    api.NewCloudError(code,reason,target,fmt,args...) -> api.NewCloudError(code,reason,target,fmt.Sprintf(fmt,args...))
// 4. Calls that provided a dynamic, pre-formatted string with arguments have their arguments brought in:
//    api.NewCloudError(code,reason,target,fmt.Sprintf(fmt,args...),args2...) -> api.NewCloudError(code,reason,target,fmt.Sprintf(fmt,append(args, args2...)...))

type options struct {
	packageNames []string
}

func newOptions() *options {
	o := &options{}
	return o
}

func (o *options) addFlags(fs *flag.FlagSet) {
}

func (o *options) complete(args []string) error {
	o.packageNames = args
	return nil
}

func (o *options) validate() error {
	if len(o.packageNames) == 0 {
		return errors.New("packages to scan are required arguments")
	}
	return nil
}

func main() {
	o := newOptions()
	o.addFlags(flag.CommandLine)
	flag.Parse()
	if err := o.complete(flag.Args()); err != nil {
		logrus.WithError(err).Fatal("could not complete options")
	}
	if err := o.validate(); err != nil {
		logrus.WithError(err).Fatal("invalid options")
	}
	if err := rewrite(o.packageNames); err != nil {
		logrus.WithError(err).Fatal("failed to re-write source")
	}
}

func rewrite(packageNames []string) error {
	pkgs, err := decorator.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Tests: true,
	}, packageNames...)
	if err != nil {
		return fmt.Errorf("failed to load source: %w", err)
	}
	for _, pkg := range pkgs {
		restorer := decorator.NewRestorerWithImports(pkg.PkgPath, gopackages.New(""))
		fileRestorer := restorer.FileRestorer()
		for i, file := range pkg.Syntax {
			filePath := pkg.CompiledGoFiles[i]
			dir := pkg.Dir
			if pkg.Module != nil { // even though we ask for modules, if we run on a single file we don't get them
				dir = pkg.Module.Dir
			}
			relPath, err := filepath.Rel(dir, filePath)
			if err != nil {
				return fmt.Errorf("should not happen: could not find relative path to %s from %s", filePath, pkg.Dir)
			}

			var shouldSkip bool
			for _, dec := range file.Decs.Start.All() {
				if dec == "// +aro-code-generator:skip" {
					shouldSkip = true
					break
				}
			}

			logrus.WithFields(logrus.Fields{"file": relPath}).Info("Considering file.")
			if shouldSkip || relPath == "pkg/api/error.go" {
				logrus.WithFields(logrus.Fields{"file": relPath}).Info("Skipping file.")
				continue
			}
			var updates int
			mutatedNode := dstutil.Apply(file, nil, func(cursor *dstutil.Cursor) bool {
				for _, update := range []func(pkg *decorator.Package, cursor *dstutil.Cursor, fileRestorer *decorator.FileRestorer) int{
					wrapUnformattedArgs,
					wrapError,
				} {
					updates += update(pkg, cursor, fileRestorer)
				}
				return true
			})
			if updates > 0 {
				logrus.WithFields(logrus.Fields{"file": relPath, "updates": updates}).Info("Updating file.")
				previous, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read contents of %s: %w", filePath, err)
				}
				f, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0666)
				if err != nil {
					return fmt.Errorf("failed to open %s for writing: %w", relPath, err)
				}
				if err := fileRestorer.Fprint(f, mutatedNode.(*dst.File)); err != nil {
					if err := os.WriteFile(filePath, previous, 0666); err != nil {
						logrus.WithFields(logrus.Fields{"file": relPath}).WithError(err).Error("Failed to restore contents of file.")
					}
					return fmt.Errorf("failed to write %s: %w", relPath, err)
				}
			}
		}
	}
	return nil
}

func wrapUnformattedArgs(pkg *decorator.Package, cursor *dstutil.Cursor, _ *decorator.FileRestorer) int {
	var updated int
	switch node := cursor.Node().(type) {
	case *dst.CallExpr:
		// we need a function call to NewCloudError
		if id, ok := node.Fun.(*dst.Ident); !ok || id.Name != "NewCloudError" {
			break
		}

		if len(node.Args) < 3 {
			// this should never happen if the code compiles
			logrus.Fatalf("found %d args", len(node.Args))
		}
		if len(node.Args) == 4 {
			// simple case, we do nothing
			logrus.Infof("ignoring NewCloudError call with %d args", len(node.Args))
			break
		}
		logrus.Infof("found NewCloudError call with %d args: %#v", len(node.Args), node.Args)
		// wrap with fmt.Sprintf if we have simple args
		_, isBasic := node.Args[3].(*dst.BasicLit)
		_, isIdent := node.Args[3].(*dst.Ident)
		if isBasic || isIdent {
			logrus.Infof("replacing unwrapped call with wrapped call")
			prefix := make([]dst.Expr, 3)
			copy(prefix, node.Args[0:3])

			fmtArgs := make([]dst.Expr, len(node.Args)-3)
			copy(fmtArgs, node.Args[3:])

			cursor.Replace(&dst.CallExpr{
				Fun: node.Fun,
				Args: append(prefix, &dst.CallExpr{
					Fun: &dst.Ident{
						Path: "fmt",
						Name: "Sprintf",
					},
					Args: fmtArgs,
				}),
				Ellipsis: node.Ellipsis,
				Decs:     node.Decs,
			})
			updated += 1
		} else {
			// move extra args into existing fmt.Sprintf if they exist
			fmtCall, ok := node.Args[3].(*dst.CallExpr)
			if !ok {
				logrus.Warnf("expected CallExpr, got %T", node.Args[3])
				break
			}
			if id, ok := fmtCall.Fun.(*dst.Ident); !ok || !(id.Path != "fmt" || id.Name != "Sprintf") {
				logrus.Warnf("expected fmt.Sprintf Ident, got %T", fmtCall.Fun)
				break
			}

			logrus.Infof("adding arguments to existing fmt.Sprintf call")
			prefix := make([]dst.Expr, 3)
			copy(prefix, node.Args[0:3])

			fmtArgs := make([]dst.Expr, len(node.Args)-4)
			copy(fmtArgs, node.Args[4:])

			fmtSpec, ok := fmtCall.Args[0].(*dst.BasicLit)
			if !ok {
				logrus.Warnf("expected BasicLit, got %T", fmtCall.Args[0])
				break
			}
			for i := 0; i < len(fmtArgs); i++ {
				fmtSpec.Value += "%v"
			}

			cursor.Replace(&dst.CallExpr{
				Fun: node.Fun,
				Args: append(prefix, &dst.CallExpr{
					Fun:  fmtCall.Fun,
					Args: append([]dst.Expr{fmtSpec}, append(fmtCall.Args[1:], fmtArgs...)...),
				}),
				Ellipsis: node.Ellipsis,
				Decs:     node.Decs,
			})
			updated += 1
		}
	}
	return updated
}

func wrapError(pkg *decorator.Package, cursor *dstutil.Cursor, _ *decorator.FileRestorer) int {
	var updated int
	switch node := cursor.Node().(type) {
	case *dst.CallExpr:
		// we need a function call to WriteError
		if id, ok := node.Fun.(*dst.Ident); !ok || !(id.Name == "WriteError" && id.Path == "github.com/Azure/ARO-RP/pkg/api") {
			break
		}

		if len(node.Args) < 4 {
			// this should never happen if the code compiles
			logrus.Fatalf("found %d args", len(node.Args))
		}
		if len(node.Args) == 5 {
			// simple case, we do nothing
			logrus.Infof("ignoring WriteError call with %d args", len(node.Args))
			break
		}
		logrus.Infof("found WriteError call with %d args: %#v", len(node.Args), node.Args)
		// wrap with fmt.Sprintf if we have simple args
		_, isBasic := node.Args[4].(*dst.BasicLit)
		_, isIdent := node.Args[4].(*dst.Ident)
		if isBasic || isIdent {
			logrus.Infof("replacing unwrapped call with wrapped call")
			prefix := make([]dst.Expr, 4)
			copy(prefix, node.Args[0:4])

			fmtArgs := make([]dst.Expr, len(node.Args)-4)
			copy(fmtArgs, node.Args[4:])

			cursor.Replace(&dst.CallExpr{
				Fun: node.Fun,
				Args: append(prefix, &dst.CallExpr{
					Fun: &dst.Ident{
						Path: "fmt",
						Name: "Sprintf",
					},
					Args: fmtArgs,
				}),
				Ellipsis: node.Ellipsis,
				Decs:     node.Decs,
			})
			updated += 1
		} else {
			// move extra args into existing fmt.Sprintf if they exist
			fmtCall, ok := node.Args[4].(*dst.CallExpr)
			if !ok {
				logrus.Warnf("expected CallExpr, got %T", node.Args[4])
				break
			}
			if id, ok := fmtCall.Fun.(*dst.Ident); !ok || !(id.Path != "fmt" || id.Name != "Sprintf") {
				logrus.Warnf("expected fmt.Sprintf Ident, got %T", fmtCall.Fun)
				break
			}

			logrus.Infof("adding arguments to existing fmt.Sprintf call")
			prefix := make([]dst.Expr, 4)
			copy(prefix, node.Args[0:4])

			fmtArgs := make([]dst.Expr, len(node.Args)-5)
			copy(fmtArgs, node.Args[5:])

			fmtSpec, ok := fmtCall.Args[0].(*dst.BasicLit)
			if !ok {
				logrus.Warnf("expected BasicLit, got %T", fmtCall.Args[0])
				break
			}
			for i := 0; i < len(fmtArgs); i++ {
				fmtSpec.Value += "%v"
			}

			cursor.Replace(&dst.CallExpr{
				Fun: node.Fun,
				Args: append(prefix, &dst.CallExpr{
					Fun:  fmtCall.Fun,
					Args: append([]dst.Expr{fmtSpec}, append(fmtCall.Args[1:], fmtArgs...)...),
				}),
				Ellipsis: node.Ellipsis,
				Decs:     node.Decs,
			})
			updated += 1
		}
	}
	return updated
}
