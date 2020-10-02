package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"sort"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func compactVerbs(in []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, 0, len(in))
	m := map[schema.GroupResource]map[string]struct{}{}

	for _, r := range in {
		if len(r.NonResourceURLs) > 0 ||
			len(r.ResourceNames) > 0 ||
			len(r.APIGroups) != 1 ||
			len(r.Resources) != 1 {
			out = append(out, r)
			continue
		}

		k := schema.GroupResource{Group: r.APIGroups[0], Resource: r.Resources[0]}
		for _, v := range r.Verbs {
			if m[k] == nil {
				m[k] = map[string]struct{}{}
			}
			m[k][v] = struct{}{}
		}
	}

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

func compactResources(in []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, 0, len(in))
	type groupVerbs struct {
		Group string
		Verbs string
	}
	m := map[groupVerbs]map[string]struct{}{}

	for _, r := range in {
		if len(r.NonResourceURLs) > 0 ||
			len(r.ResourceNames) > 0 ||
			len(r.APIGroups) != 1 {
			out = append(out, r)
			continue
		}

		k := groupVerbs{Group: r.APIGroups[0], Verbs: strings.Join(r.Verbs, "/")}
		for _, r := range r.Resources {
			if m[k] == nil {
				m[k] = map[string]struct{}{}
			}
			m[k][r] = struct{}{}
		}
	}

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
