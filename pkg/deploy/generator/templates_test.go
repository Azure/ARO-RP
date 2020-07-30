package generator

import (
	"reflect"
	"testing"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestConditionStanza(t *testing.T) {
	for _, tt := range []struct {
		name       string
		parameters []string
		want       string
	}{
		{
			name:       "single parameter",
			parameters: []string{"a"},
			want:       "[parameters('a')]",
		},
		{
			name:       "multiple parameters",
			parameters: []string{"a", "b"},
			want:       "[and(parameters('a'),parameters('b'))]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := generator{production: true}

			got := g.conditionStanza(tt.parameters...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%#v", got)
			}
		})
	}

}
