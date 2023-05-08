package token

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "testing"

func TestGetObjectId(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		want    string
		wantErr bool
	}{
		{
			name:    "Can extract oid from a valid token",
			token:   "eyJhbGciOiJIUzI1NiJ9.eyJJc3N1ZXIiOiJJc3N1ZXIiLCJvaWQiOiJub3dheXRoaXNpc2FyZWFsYXBwIiwiZXhwIjoxNjgwODQ2MDI2LCJpYXQiOjE2ODA4NDYwMjZ9.GQxPJbMJYhrXK1YlWUXR_5IpBlvkv9kEdX_Z_vJRxsU",
			want:    "nowaythisisarealapp",
			wantErr: false,
		},
		{
			name:    "Return an error when given an invalid jwt",
			token:   "invalid",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetObjectId(tt.token)
			t.Log(got, err)
			if got != tt.want {
				t.Errorf("Got oid: %q, want %q", got, tt.want)
			}
			if tt.wantErr && err == nil {
				t.Errorf("Expect an error but got nothing")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expect no error but got one")
			}
		})
	}
}
