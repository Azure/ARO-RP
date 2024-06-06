package api_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

func TestAccessToken(t *testing.T) {
	testCases := []struct {
		name           string
		clusterID      string
		token          string
		expAccessToken string
	}{
		{
			name:           "Test AccessToken String",
			clusterID:      "123",
			token:          "abc",
			expAccessToken: "123:abc",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			accessToken := api.NewAccessToken(tc.clusterID, tc.token)
			if accessToken.String() != tc.expAccessToken {
				t.Errorf("AccessToken got %v, expect %v", accessToken.String(), tc.expAccessToken)
			}
		})
	}
}
