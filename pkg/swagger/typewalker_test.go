package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"go/ast"
	"reflect"
	"testing"
)

func TestGetNodeField(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		nodes    []ast.Node
		expected *ast.Field
		ok       bool
	}{
		{
			name:  "fewer than two nodes",
			nodes: []ast.Node{},
			ok:    false,
		},
		{
			name: "second node is a field",
			nodes: []ast.Node{
				nil,
				&ast.Field{
					Names: []*ast.Ident{ast.NewIdent("StatusCode")},
				},
			},
			expected: &ast.Field{
				Names: []*ast.Ident{ast.NewIdent("StatusCode")},
			},
			ok: true,
		},
		{
			name: "third node is a field",
			nodes: []ast.Node{
				nil,
				nil,
				&ast.Field{
					Names: []*ast.Ident{ast.NewIdent("CloudErrorBody")},
				},
			},
			expected: &ast.Field{
				Names: []*ast.Ident{ast.NewIdent("CloudErrorBody")},
			},
			ok: true,
		},
		{
			name:  "no field in nodes",
			nodes: []ast.Node{nil, nil, nil},
			ok:    false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, ok := getNodeField(tt.nodes)
			if ok != tt.ok {
				t.Errorf("expected ok to be %t, got %t", tt.ok, ok)
			}
			if !reflect.DeepEqual(field, tt.expected) {
				t.Errorf("expected field to be %+v, got %+v", tt.expected, field)
			}
		})
	}
}
