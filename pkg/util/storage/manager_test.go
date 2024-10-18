package storage

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestGetCorrectErrWhenTooManyRequests(t *testing.T) {
	for _, tt := range []struct {
		name    string
		err     error
		wantErr string
	}{
		{
			name: "No Error",
		},
		{
			name: "Too Many Requests Error",
			err: &azcore.ResponseError{
				ErrorCode: fmt.Sprintf("%d", http.StatusTooManyRequests),
			},
			wantErr: `Missing RawResponse
--------------------------------------------------------------------------------
ERROR CODE: 429
--------------------------------------------------------------------------------
`,
		},
		{
			name:    "Not a azcore.ResponseError",
			err:     errors.New("Test Error"),
			wantErr: "Test Error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resErr := getCorrectErrWhenTooManyRequests(tt.err)
			if tt.err != nil {
				if resErr.Error() != tt.wantErr {
					t.Fatalf("Expected %s, got %s", tt.wantErr, resErr.Error())
				}
			} else {
				if resErr != nil {
					t.Fatalf("Response Error is not nil")
				}
			}
		})
	}
}
