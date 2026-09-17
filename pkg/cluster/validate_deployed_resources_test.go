package cluster

import (
	"testing"
)

func TestValidatePropertiesRecursive(t *testing.T) {
	tests := []struct {
		name          string
		actualProps   map[string]interface{}
		expectedProps map[string]interface{}
		resourceName  string
		prefix        string
		wantMismatches []propMismatch
	}{
		{
			name: "exact match",
			actualProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
			},
			expectedProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
			},
			resourceName:   "testAccount",
			wantMismatches: []propMismatch{},
		},
		{
			name: "value mismatch",
			actualProps: map[string]interface{}{
				"publicNetworkAccess": "Disabled",
			},
			expectedProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "publicNetworkAccess",
					Expected: "Enabled",
					Actual:   "Disabled",
				},
			},
		},
		{
			name: "missing key in actual",
			actualProps: map[string]interface{}{
				"other": "value",
			},
			expectedProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
			},
			resourceName:   "testAccount",
			wantMismatches: []propMismatch{},
		},
		{
			name: "nested map match",
			actualProps: map[string]interface{}{
				"settings": map[string]interface{}{
					"timeout": float64(30),
					"retries": float64(3),
				},
			},
			expectedProps: map[string]interface{}{
				"settings": map[string]interface{}{
					"timeout": float64(30),
				},
			},
			resourceName:   "testAccount",
			wantMismatches: []propMismatch{},
		},
		{
			name: "nested map value mismatch",
			actualProps: map[string]interface{}{
				"settings": map[string]interface{}{
					"timeout": float64(60),
				},
			},
			expectedProps: map[string]interface{}{
				"settings": map[string]interface{}{
					"timeout": float64(30),
				},
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "settings.timeout",
					Expected: float64(30),
					Actual:   float64(60),
				},
			},
		},
		{
			name: "deeply nested mismatch",
			actualProps: map[string]interface{}{
				"config": map[string]interface{}{
					"db": map[string]interface{}{
						"host": "localhost",
						"port": float64(5433),
					},
				},
			},
			expectedProps: map[string]interface{}{
				"config": map[string]interface{}{
					"db": map[string]interface{}{
						"port": float64(5432),
					},
				},
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "config.db.port",
					Expected: float64(5432),
					Actual:   float64(5433),
				},
			},
		},
		{
			name: "multiple mismatches",
			actualProps: map[string]interface{}{
				"publicNetworkAccess": "Disabled",
				"settings": map[string]interface{}{
					"timeout": float64(60),
				},
			},
			expectedProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
				"settings": map[string]interface{}{
					"timeout": float64(30),
				},
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "publicNetworkAccess",
					Expected: "Enabled",
					Actual:   "Disabled",
				},
				{
					Resource: "testAccount",
					Property: "settings.timeout",
					Expected: float64(30),
					Actual:   float64(60),
				},
			},
		},
		{
			name: "nil value mismatch",
			actualProps: map[string]interface{}{
				"publicNetworkAccess": nil,
			},
			expectedProps: map[string]interface{}{
				"publicNetworkAccess": "Enabled",
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "publicNetworkAccess",
					Expected: "Enabled",
					Actual:   nil,
				},
			},
		},
		{
			name: "with prefix",
			actualProps: map[string]interface{}{
				"timeout": float64(60),
			},
			expectedProps: map[string]interface{}{
				"timeout": float64(30),
			},
			resourceName: "testAccount",
			prefix:       "settings",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "settings.timeout",
					Expected: float64(30),
					Actual:   float64(60),
				},
			},
		},
		{
			name: "nested map not matching actual type (string instead of map)",
			actualProps: map[string]interface{}{
				"settings": "not a map",
			},
			expectedProps: map[string]interface{}{
				"settings": map[string]interface{}{
					"timeout": float64(30),
				},
			},
			resourceName: "testAccount",
			wantMismatches: []propMismatch{
				{
					Resource: "testAccount",
					Property: "settings",
					Expected: map[string]interface{}{"timeout": float64(30)},
					Actual:   "not a map",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mismatches []propMismatch
			validatePropertiesRecursive(tt.actualProps, tt.expectedProps, tt.resourceName, tt.prefix, &mismatches)

			if len(mismatches) != len(tt.wantMismatches) {
				t.Errorf("got %d mismatches, want %d", len(mismatches), len(tt.wantMismatches))
			}

			for i, m := range mismatches {
				if i >= len(tt.wantMismatches) {
					t.Errorf("unexpected mismatch: %+v", m)
					continue
				}
				want := tt.wantMismatches[i]
				if m.Resource != want.Resource || m.Property != want.Property || m.Expected != want.Expected || m.Actual != want.Actual {
					t.Errorf("mismatch %d: got %+v, want %+v", i, m, want)
				}
			}
		})
	}
}
