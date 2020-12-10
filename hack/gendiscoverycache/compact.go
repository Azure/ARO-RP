package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"sort"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Kubernetes RBAC uses the cross-product of PolicyRules stored in Role or
// ClusterRole resources, thus enabling the PolicyRules to be stored more
// compactly.  For example:
//
// rules:
// - apiGroups: [ group1 ]
//   resources: [ resource1 ]
//   verbs: [ verb1 ]
// - apiGroups: [ group1 ]
//   resources: [ resource1 ]
//   verbs: [ verb2 ]
// - apiGroups: [ group1 ]
//   resources: [ resource2 ]
//   verbs: [ verb1 ]
// - apiGroups: [ group1 ]
//   resources: [ resource2 ]
//   verbs: [ verb2 ]
//
// is equivalent to:
//
// rules:
// - apiGroups: [ group1 ]
//   resources: [ resource1, resource2 ]
//   verbs: [ verb1, verb2 ]
//
//
// This file contains functions which can compact slices of individual
// PolicyRules according to some simple rules.  We do not attempt to cover all
// possible simplifications.

// compactVerbs compacts simple PolicyRules which differ only in their verbs
// into a single PolicyRule with all the verbs combined.
func compactVerbs(in []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, 0, len(in))
	m := map[schema.GroupResource]map[string]struct{}{}

	// 1. Collate matching simple PolicyRules into the map.
	for _, r := range in {
		if len(r.NonResourceURLs) > 0 ||
			len(r.ResourceNames) > 0 ||
			len(r.APIGroups) != 1 ||
			len(r.Resources) != 1 {
			// rule too complex for us to deal with - emit and continue
			out = append(out, r)
			continue
		}

		// add rule to map so we can compact verbs
		k := schema.GroupResource{Group: r.APIGroups[0], Resource: r.Resources[0]}
		for _, v := range r.Verbs {
			if m[k] == nil {
				m[k] = map[string]struct{}{}
			}
			m[k][v] = struct{}{}
		}
	}

	// 2. Walk the map emitting a single PolicyRule for each key with the verbs
	// combined.
	for gr, verbs := range m {
		pr := &rbacv1.PolicyRule{
			APIGroups: []string{gr.Group},
			Resources: []string{gr.Resource},
		}

		for v := range verbs {
			pr.Verbs = append(pr.Verbs, v)
		}
		sort.Strings(pr.Verbs)

		out = append(out, *pr)
	}

	return out
}

// compactResources compacts simple PolicyRules which differ only in their
// resources into a single PolicyRule with all the resources combined.
func compactResources(in []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, 0, len(in))
	type groupVerbs struct {
		Group string
		Verbs string
	}
	m := map[groupVerbs]map[string]struct{}{}

	// 1. Collate matching simple PolicyRules into the map.
	for _, r := range in {
		if len(r.NonResourceURLs) > 0 ||
			len(r.ResourceNames) > 0 ||
			len(r.APIGroups) != 1 {
			// rule too complex for us to deal with - emit and continue
			out = append(out, r)
			continue
		}

		// add rule to map so we can compact resources
		k := groupVerbs{Group: r.APIGroups[0], Verbs: strings.Join(r.Verbs, "/")}
		for _, r := range r.Resources {
			if m[k] == nil {
				m[k] = map[string]struct{}{}
			}
			m[k][r] = struct{}{}
		}
	}

	// 2. Walk the map emitting a single PolicyRule for each key with the
	// resources combined.
	for gv, resources := range m {
		pr := &rbacv1.PolicyRule{
			APIGroups: []string{gv.Group},
			Verbs:     strings.Split(gv.Verbs, "/"),
		}

		for r := range resources {
			pr.Resources = append(pr.Resources, r)
		}
		sort.Strings(pr.Resources)

		out = append(out, *pr)
	}

	return out
}

func compactRules(rules []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	rules = compactVerbs(rules)
	rules = compactResources(rules)

	return rules
}
