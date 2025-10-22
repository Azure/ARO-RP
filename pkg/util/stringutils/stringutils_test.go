package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestIndentLines(t *testing.T) {
	in := "hello"
	expected := "  hello"
	out := IndentLines(in, "  ")
	assert.Equal(t, expected, out)

	in = "hello\nthere\nfriends"
	expected = "  hello\n  there\n  friends"
	out = IndentLines(in, "  ")
	assert.Equal(t, expected, out)
}

func TestGroupsUnion(t *testing.T) {
	for _, tt := range []struct {
		name string
		as   []string
		bs   []string
		want []string
	}{
		{
			name: "Empty arrays Union",
			as:   []string{},
			bs:   []string{},
			want: []string{},
		},
		{
			name: "First array empty",
			as:   []string{},
			bs:   []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Second array empty",
			as:   []string{"a", "b", "c"},
			bs:   []string{},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Identical arrays Union",
			as:   []string{"a", "b", "c"},
			bs:   []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Partial overlap arrays Union",
			as:   []string{"a", "b", "c"},
			bs:   []string{"b", "c", "d", "e"},
			want: []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "No overlap arrays Union",
			as:   []string{"a", "b", "c"},
			bs:   []string{"d", "e", "f"},
			want: []string{"a", "b", "c", "d", "e", "f"},
		},
		{
			name: "Nil arrays Union",
			as:   nil,
			bs:   nil,
			want: []string{},
		},
		{
			name: "First array nil",
			as:   nil,
			bs:   []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "Second array nil",
			as:   []string{"a", "b"},
			bs:   nil,
			want: []string{"a", "b"},
		},
		{
			name: "Arrays with duplicates within themselves",
			as:   []string{"a", "a", "b", "c"},
			bs:   []string{"b", "b", "d"},
			want: []string{"d", "c", "b", "a"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupsUnion(tt.as, tt.bs)
			// Since map iteration order is non-deterministic, we need to compare as sets
			assert.ElementsMatch(t, tt.want, result, "Union result should contain all unique elements from both arrays")
		})
	}
}
