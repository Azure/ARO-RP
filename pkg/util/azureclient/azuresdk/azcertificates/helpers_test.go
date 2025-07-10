package azcertificates

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func errorInfoContainsTrue(e *azcertificates.ErrorInfo, substr string) bool  { return true }
func errorInfoContainsFalse(e *azcertificates.ErrorInfo, substr string) bool { return false }

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
			op:                azcertificates.CertificateOperation{Status: pointerutils.ToPtr("inProgress")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    false,
			expectedError:     nil,
		},
		{
			name:              "Completed",
			op:                azcertificates.CertificateOperation{Status: pointerutils.ToPtr("completed")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    true,
			expectedError:     nil,
		},
		{
			name:              "Failed",
			op:                azcertificates.CertificateOperation{Status: pointerutils.ToPtr("failed")},
			errorInfoContains: errorInfoContainsFalse,
			expectedResult:    false,
			expectedError:     fmt.Errorf("certificateOperation %s: Error %v", "failed", nil),
		},
		{
			name:              "FailedCanRetry",
			op:                azcertificates.CertificateOperation{Status: pointerutils.ToPtr("failed")},
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
