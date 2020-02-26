package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestIsTerminal(t *testing.T) {
	for _, tt := range []struct {
		name  string
		want  bool
		state ProvisioningState
	}{
		{
			name:  "Success is Terminal",
			want:  true,
			state: ProvisioningStateSucceeded,
		},
		{
			name:  "Failed is Terminal",
			want:  true,
			state: ProvisioningStateFailed,
		},
		{
			name:  "Creating is Non-Terminal",
			want:  false,
			state: ProvisioningStateCreating,
		},
		{
			name:  "Updating is Non-Terminal",
			want:  false,
			state: ProvisioningStateUpdating,
		},
		{
			name:  "Deleting is Non-Terminal",
			want:  false,
			state: ProvisioningStateDeleting,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsTerminal() != tt.want {
				t.Fatalf("%s isTerminal wants != %t", tt.state, tt.want)
			}
		})
	}
}
