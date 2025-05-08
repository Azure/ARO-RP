package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestCompactVerbs(t *testing.T) {
	nonCompacted := []rbacv1.PolicyRule{
		{
			NonResourceURLs: []string{"/foo"},
			Verbs:           []string{"create"},
		},
		{
			NonResourceURLs: []string{"/foo"},
			Verbs:           []string{"get"},
		},
		{
			Resources:     []string{"resource"},
			ResourceNames: []string{"foo"},
			Verbs:         []string{"create"},
		},
		{
			Resources:     []string{"resource"},
			ResourceNames: []string{"foo"},
			Verbs:         []string{"get"},
		},
		{
			APIGroups: []string{"group1", "group2"},
			Resources: []string{"foo"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"group1", "group2"},
			Resources: []string{"foo"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"b"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"a"},
			Verbs:     []string{"create"},
		},
	}

	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"b"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"b"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"b"},
			Verbs:     []string{"list"},
		},
	}
	rules = append(rules, nonCompacted...)

	want := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"create", "get"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"b"},
			Verbs:     []string{"create", "list"},
		},
	}
	want = append(want, nonCompacted...)
	sort.Slice(want, sorter(want))

	rules = compactVerbs(rules)
	sort.Slice(rules, sorter(rules))
	if !reflect.DeepEqual(rules, want) {
		t.Error(rules)
	}
}

func TestCompactResources(t *testing.T) {
	nonCompacted := []rbacv1.PolicyRule{
		{
			NonResourceURLs: []string{"/foo"},
			Verbs:           []string{"create"},
		},
		{
			NonResourceURLs: []string{"/foo"},
			Verbs:           []string{"get"},
		},
		{
			Resources:     []string{"resource"},
			ResourceNames: []string{"foo"},
			Verbs:         []string{"create"},
		},
		{
			Resources:     []string{"resource"},
			ResourceNames: []string{"foo"},
			Verbs:         []string{"get"},
		},
		{
			APIGroups: []string{"group1", "group2"},
			Resources: []string{"foo"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"group1", "group2"},
			Resources: []string{"bar"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"c"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"c"},
			Verbs:     []string{"create", "update", "delete"},
		},
	}

	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"create", "update"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a"},
			Verbs:     []string{"create", "update"},
		},
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"b"},
			Verbs:     []string{"create", "update"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"a"},
			Verbs:     []string{"create", "list"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"a"},
			Verbs:     []string{"create", "list"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"b"},
			Verbs:     []string{"create", "list"},
		},
	}
	rules = append(rules, nonCompacted...)

	want := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"groupA"},
			Resources: []string{"a", "b"},
			Verbs:     []string{"create", "update"},
		},
		{
			APIGroups: []string{"groupB"},
			Resources: []string{"a", "b"},
			Verbs:     []string{"create", "list"},
		},
	}
	want = append(want, nonCompacted...)
	sort.Slice(want, sorter(want))

	rules = compactResources(rules)
	sort.Slice(rules, sorter(rules))
	if !reflect.DeepEqual(rules, want) {
		t.Error(rules)
	}
}

func sorter(rules []rbacv1.PolicyRule) func(i, j int) bool {
	return func(i, j int) bool {
		// weird but works
		ii, err := json.Marshal(rules[i])
		if err != nil {
			panic(err)
		}

		jj, err := json.Marshal(rules[j])
		if err != nil {
			panic(err)
		}

		return bytes.Compare(ii, jj) < 0
	}
}
