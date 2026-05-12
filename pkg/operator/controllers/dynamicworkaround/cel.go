package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

// CEL variable names exposed to expressions. Keeping these as package-level
// constants makes it easier to add new ones without sprinkling string literals
// across the env setup and the input map.
const (
	celVarClusterVersion      = "clusterVersion"
	celVarIPSecMode           = "ipsecMode"
	celVarRegion              = "region"
	celVarArchitectureVersion = "architectureVersion"
)

// celEvalTimeout caps the wall-clock time a single CEL expression may run.
// 250ms is generous for the simple boolean predicates we expect; anything
// approaching this limit indicates a pathological expression and should be
// caught at catalog-review time, not at runtime.
const celEvalTimeout = 250 * time.Millisecond

// celEnv is the shared CEL environment used to compile every predicate
// expression. It is built once at package init and never mutated.
var (
	celEnv     *cel.Env
	celEnvErr  error
	celEnvOnce sync.Once
)

// getCELEnv returns the lazily-constructed CEL environment. We initialise on
// first use rather than in init() so that a build-time CEL misconfiguration
// surfaces as a test failure on the predicate path rather than a confusing
// panic during operator startup.
func getCELEnv() (*cel.Env, error) {
	celEnvOnce.Do(func() {
		celEnv, celEnvErr = cel.NewEnv(
			cel.Variable(celVarClusterVersion, cel.StringType),
			cel.Variable(celVarIPSecMode, cel.StringType),
			cel.Variable(celVarRegion, cel.StringType),
			cel.Variable(celVarArchitectureVersion, cel.IntType),

			// versionAtLeast(facts, target) returns true iff `facts` is a
			// semver-parseable string and ≥ target. We deliberately make this
			// the only domain-specific helper: simple string comparison would
			// be lexicographic and misorder versions like "4.10.0" vs "4.9.0".
			cel.Function("versionAtLeast",
				cel.Overload("versionAtLeast_string_string",
					[]*cel.Type{cel.StringType, cel.StringType},
					cel.BoolType,
					cel.BinaryBinding(versionAtLeastImpl),
				),
			),
			// versionLessThan(facts, target) returns true iff `facts` is a
			// semver-parseable string and < target. Pairs with versionAtLeast
			// to make the common "I want range [a, b)" expression natural.
			cel.Function("versionLessThan",
				cel.Overload("versionLessThan_string_string",
					[]*cel.Type{cel.StringType, cel.StringType},
					cel.BoolType,
					cel.BinaryBinding(versionLessThanImpl),
				),
			),
		)
	})
	return celEnv, celEnvErr
}

// versionAtLeastImpl implements the CEL `versionAtLeast` function. CEL passes
// the raw operands as ref.Val; we unwrap them, parse both sides with the same
// version helper the static predicate fields use, and produce a Bool back.
//
// An unparseable input is treated as a runtime CEL error (rather than false)
// so catalog authors get a loud signal during testing instead of silent
// no-match.
func versionAtLeastImpl(lhs, rhs ref.Val) ref.Val {
	left, ok := lhs.Value().(string)
	if !ok {
		return types.NewErr("versionAtLeast: lhs must be string, got %T", lhs.Value())
	}
	right, ok := rhs.Value().(string)
	if !ok {
		return types.NewErr("versionAtLeast: rhs must be string, got %T", rhs.Value())
	}
	if left == "" {
		// clusterVersion is "" when we couldn't determine it. Return false so
		// version-gated predicates fail closed rather than match by accident.
		return types.Bool(false)
	}
	lv, err := version.ParseVersion(left)
	if err != nil {
		return types.NewErr("versionAtLeast: lhs %q: %v", left, err)
	}
	rv, err := version.ParseVersion(right)
	if err != nil {
		return types.NewErr("versionAtLeast: rhs %q: %v", right, err)
	}
	return types.Bool(!lv.Lt(rv))
}

// versionLessThanImpl is the mirror of versionAtLeastImpl for the < operator.
func versionLessThanImpl(lhs, rhs ref.Val) ref.Val {
	left, ok := lhs.Value().(string)
	if !ok {
		return types.NewErr("versionLessThan: lhs must be string, got %T", lhs.Value())
	}
	right, ok := rhs.Value().(string)
	if !ok {
		return types.NewErr("versionLessThan: rhs must be string, got %T", rhs.Value())
	}
	if left == "" {
		return types.Bool(false)
	}
	lv, err := version.ParseVersion(left)
	if err != nil {
		return types.NewErr("versionLessThan: lhs %q: %v", left, err)
	}
	rv, err := version.ParseVersion(right)
	if err != nil {
		return types.NewErr("versionLessThan: rhs %q: %v", right, err)
	}
	return types.Bool(lv.Lt(rv))
}

// compileCEL parses `expr`, checks that it produces a bool, and returns the
// compiled Program. Returns an error if the expression is syntactically
// invalid, references unknown variables, or has a non-bool result type.
//
// This is the single entry point used by both Catalog.Validate (which fails
// the whole catalog on a bad expression) and Predicate.evalCEL (which runs
// the program against ClusterFacts at reconcile time). Compiling twice — once
// at validate, once at eval — is cheap for the catalog sizes we care about
// (≤ 64 entries) and keeps the Predicate struct free of un-marshalable state.
func compileCEL(expr string) (cel.Program, error) {
	env, err := getCELEnv()
	if err != nil {
		return nil, fmt.Errorf("build CEL env: %w", err)
	}
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile CEL: %w", issues.Err())
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("CEL expression must return bool, got %s", ast.OutputType())
	}
	prg, err := env.Program(ast, cel.InterruptCheckFrequency(100))
	if err != nil {
		return nil, fmt.Errorf("build CEL program: %w", err)
	}
	return prg, nil
}

// celFactsAsInput converts ClusterFacts to the map shape CEL Program.Eval
// expects. Stringifying ClusterVersion gives us something CEL can compare via
// the versionAtLeast/versionLessThan helpers; we use "" for nil so the
// helpers fail closed (see versionAtLeastImpl).
func celFactsAsInput(facts ClusterFacts) map[string]any {
	clusterVersionStr := ""
	if facts.ClusterVersion != nil {
		clusterVersionStr = facts.ClusterVersion.String()
	}
	return map[string]any{
		celVarClusterVersion:      clusterVersionStr,
		celVarIPSecMode:           facts.IPSecMode,
		celVarRegion:              facts.Location,
		celVarArchitectureVersion: int64(facts.ArchitectureVersion),
	}
}

// runCELProgram runs an already-compiled program against facts under a hard
// timeout. Programs are produced by compileCEL once at flag-parse time and
// then reused for the lifetime of the parsed Predicates value, so this is
// the hot-path entry point.
//
// Returns the bool result, or an error if evaluation times out, fails inside
// a custom function (e.g. versionAtLeast with a bad arg), or returns a non-
// bool. The caller is expected to treat any error as "skip this workaround"
// so one bad expression doesn't block the other entries.
func runCELProgram(ctx context.Context, prg cel.Program, facts ClusterFacts) (bool, error) {
	evalCtx, cancel := context.WithTimeout(ctx, celEvalTimeout)
	defer cancel()

	out, _, err := prg.ContextEval(evalCtx, celFactsAsInput(facts))
	if err != nil {
		return false, fmt.Errorf("eval CEL: %w", err)
	}
	b, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression returned %T, want bool", out.Value())
	}
	return b, nil
}

// evalCEL is a convenience wrapper that compiles + runs in one call. It is
// used only by tests; production code goes through Predicates.Eval which
// caches compiled programs across reconciles.
func evalCEL(ctx context.Context, expr string, facts ClusterFacts) (bool, error) {
	prg, err := compileCEL(expr)
	if err != nil {
		return false, err
	}
	return runCELProgram(ctx, prg, facts)
}
