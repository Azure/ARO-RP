package azcertificates

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func errorInfoContainsTrue(e *azcertificates.ErrorInfo, substr string) bool  { return true }
func errorInfoContainsFalse(e *azcertificates.ErrorInfo, substr string) bool { return false }

func ptr(s string) *string {
	return &s
}

func TestCheckOperation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	tests := []struct {
		name              string
		op                azcertificates.CertificateOperation
		errorInfoContains func(e *azcertificates.ErrorInfo, substr string) bool
		expectedResult    bool
		expectedError     error
	}{
		{
			name:              "In Progress",
			op:                azcertificates.CertificateOperation{Status: ptr("inProgress")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    false,
			expectedError:     nil,
		},
		{
			name:              "Completed",
			op:                azcertificates.CertificateOperation{Status: ptr("completed")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    true,
			expectedError:     nil,
		},
		{
			name:              "Failed",
			op:                azcertificates.CertificateOperation{Status: ptr("failed")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    false,
			expectedError:     fmt.Errorf("certificateOperation %s: Error %v", "failed", nil),
		},
		{
			name:              "FailedCanRetry",
			op:                azcertificates.CertificateOperation{Status: ptr("failed")},
			errorInfoContains: errorInfoContainsTrue,
			expectedResult:    false,
			expectedError:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errorInfoContains = tc.errorInfoContains
			result, err := checkOperation(tc.op, log)
			assert.Equal(t, tc.expectedResult, result)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
