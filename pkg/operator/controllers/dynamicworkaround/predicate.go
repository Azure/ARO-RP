package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/cel"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

// MaxExprLength caps the size of a single CEL expression. CEL itself has no
// length limit and a pathological expression would slow down compile + run
// time; 4096 chars is far more than any sensible boolean predicate needs and
// is a defence against authoring mistakes, not a security boundary.
const MaxExprLength = 4096

// MaxPredicatesBytes caps the raw size of the predicates flag value. The flag
// is just JSON, so a value over this size is almost certainly an accident and
// we'd rather log+reject than try to parse it.
const MaxPredicatesBytes = 64 * 1024

// ClusterFacts is everything the predicate evaluator needs to know about the
// local cluster. Built once per reconcile to keep evaluation cheap and to
// keep all the awkward "where does this come from?" plumbing in the controller.
type ClusterFacts struct {
	// ClusterVersion is the cluster's current OpenShift version (e.g. 4.16.7).
	// nil is treated as "unknown" by the CEL evaluator: the versionAtLeast /
	// versionLessThan helpers fail closed in that case so we never apply
	// version-gated workarounds with bad data.
	ClusterVersion version.Version

	// IPSecMode is the literal string from
	// network.spec.defaultNetwork.ovnKubernetesConfig.ipsecConfig.mode.
	// Empty string ("") means the Network CR was missing the field; an
	// expression that wants to match that case writes `ipsecMode == ""`.
	IPSecMode string

	// Location is the cluster Azure region (e.g. "eastus"), lower-case.
	Location string

	// ArchitectureVersion is 1 or 2.
	ArchitectureVersion int
}

// Predicates is the per-cluster opt-in configuration: a map from a catalog
// workaround Name to a compiled CEL boolean expression. A catalog entry
// applies on this cluster iff its name appears in this map AND the expression
// returns true for the local cluster's facts.
//
// Predicates is built by parsePredicates from the value of the
// `aro.dynamicworkaround.predicates` operator flag (a JSON object). The catalog
// itself does not ship predicates — gating lives entirely on the cluster
// side so the same catalog can roll out to different cohorts independently.
type Predicates map[string]cel.Program

// parsePredicates decodes the raw flag value and compiles each entry. The
// flag is a JSON object literal:
//
//	{
//	  "ipsec-mtu-fix":     "ipsecMode == \"Full\" && region == \"eastus\"",
//	  "kernel-quirk-4-16": "versionAtLeast(clusterVersion, \"4.16.0\") && versionLessThan(clusterVersion, \"4.17.0\")"
//	}
//
// Returns nil (empty map) for the empty / absent flag — equivalent to "no
// workarounds enabled on this cluster". Returns an error if the JSON is
// malformed or any single expression fails to compile; the caller is
// expected to treat a parse error as "skip applying anything this reconcile"
// rather than tearing down existing mitigations.
//
// Compiling at parse time (rather than at every Eval) keeps reconcile-hot-
// path cost flat and surfaces bad expressions in operator logs at flag-set
// time instead of silently no-matching forever.
func parsePredicates(raw string) (Predicates, error) {
	if raw == "" {
		return Predicates{}, nil
	}
	if len(raw) > MaxPredicatesBytes {
		return nil, fmt.Errorf("predicates flag is %d bytes; cap is %d", len(raw), MaxPredicatesBytes)
	}

	var rawMap map[string]string
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		return nil, fmt.Errorf("decode predicates flag: %w", err)
	}

	out := make(Predicates, len(rawMap))
	for name, expr := range rawMap {
		if !nameRegex.MatchString(name) {
			return nil, fmt.Errorf("predicate key %q is not a valid workaround name", name)
		}
		if expr == "" {
			return nil, fmt.Errorf("predicate %q: empty expression", name)
		}
		if len(expr) > MaxExprLength {
			return nil, fmt.Errorf("predicate %q: expression exceeds %d characters", name, MaxExprLength)
		}
		prg, err := compileCEL(expr)
		if err != nil {
			return nil, fmt.Errorf("predicate %q: %w", name, err)
		}
		out[name] = prg
	}
	return out, nil
}

// Eval looks up the predicate for `name` and evaluates it against `facts`.
//
// Return semantics:
//
//	matched=true,  hasPredicate=true,  err=nil   — workaround should apply
//	matched=false, hasPredicate=true,  err=nil   — predicate evaluated false
//	matched=false, hasPredicate=false, err=nil   — no predicate configured;
//	                                               workaround is disabled here
//	matched=false, hasPredicate=true,  err!=nil  — evaluation failed
//	                                               (e.g. bad version arg)
//
// Splitting `matched` from `hasPredicate` lets the caller log the two cases
// distinctly: a missing predicate is the normal "this cluster has not opted
// in" path, whereas a false predicate is interesting (the operator opted in
// but the cluster facts didn't match).
func (p Predicates) Eval(ctx context.Context, name string, facts ClusterFacts) (matched bool, hasPredicate bool, err error) {
	prg, ok := p[name]
	if !ok {
		return false, false, nil
	}
	matched, err = runCELProgram(ctx, prg, facts)
	return matched, true, err
}
