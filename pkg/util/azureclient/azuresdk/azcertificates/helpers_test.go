package azcertificates

import (
    "fmt"
    "testing"

    "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
)

// Create a helper function for constructing *azcertificates.ErrorInfo
func createErrorInfo(message string) *azcertificates.ErrorInfo {
    return &azcertificates.ErrorInfo{
        Code:    "0", // Replace with an appropriate error code
        data: 	 &message,
    }
}


func TestCheckOperation(t *testing.T) {
    log := logrus.NewEntry(logrus.New())

    tests := []struct {
        name           string
        op             azcertificates.CertificateOperation
        expectedResult bool
        expectedError  error
    }{
        {
            name:           "Status nil",
            op:             azcertificates.CertificateOperation{Status: nil},
            expectedResult: false,
            expectedError:  fmt.Errorf("operation status is nil"),
        },
        {
            name:           "In Progress",
            op:             azcertificates.CertificateOperation{Status: ptr("inProgress")},
            expectedResult: false,
            expectedError:  nil,
        },
        {
            name:           "Completed",
            op:             azcertificates.CertificateOperation{Status: ptr("completed")},
            expectedResult: true,
            expectedError:  nil,
        },
        {
            name: "Failed",
            op: azcertificates.CertificateOperation{
                Status: ptr("failed"),
                Error:  createErrorInfo("[Status:Failed] Operation failed"),
            },
            expectedResult: false,
            expectedError:  fmt.Errorf("certificateOperation %s: Error %w", "failed", "[Status:Failed] Operation failed"),
        },
        {
            name: "FailedCanRetry",
            op: azcertificates.CertificateOperation{
                Status: ptr("failed"),
                Error:  createErrorInfo("[Status:FailedCanRetry] Operation failed"),
            },
            expectedResult: false,
            expectedError:  nil,
        },
        {
            name: "Unsupported Status",
            op: azcertificates.CertificateOperation{
                Status: ptr("unknown"),
                Error:  createErrorInfo("unknown error"),
            },
            expectedResult: false,
            expectedError:  fmt.Errorf("certificateOperation unknown: Error %w", fmt.Errorf("unknown error")),
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
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

func ptr(s string) *string {
    return &s
}
