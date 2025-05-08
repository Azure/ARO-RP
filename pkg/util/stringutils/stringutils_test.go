package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
)

func TestLastTokenByte(t *testing.T) {
	result := LastTokenByte("a/b/c/d", '/')
	want := "d"
	if result != want {
		t.Errorf("want %s, got %s", want, result)
	}
}

func TestGroupsIntersect(t *testing.T) {
	for _, tt := range []struct {
		name string
		as   []string
		bs   []string
		want []string
	}{
		{
			name: "Empty array Intersection",
			as:   []string{},
			bs:   []string{},
		},
		{
			name: "Matching array Intersection",
			as:   []string{"a", "b", "c"},
			bs:   []string{"b", "a", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Partial array Intersection",
			as:   []string{"a", "b", "c"},
			bs:   []string{"d", "e", "a"},
			want: []string{"a"},
		},
		{
			name: "No array Intersection",
			as:   []string{"a", "b", "c"},
			bs:   []string{"d", "e", "f"},
		},
		{
			name: "Nil array Intersection",
			as:   []string{"a", "b", "c"},
			bs:   nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupsIntersect(tt.as, tt.bs)
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("want %s, got %s", tt.want, result)
			}
		})
	}
}
