package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
)

const (
	ErrCodeInvalidClientSecretProvided = "AADSTS7000215" // https://login.microsoftonline.com/error?code=7000215
	ErrCodeMissingRequiredParameters   = "AADSTS7000216" // https://login.microsoftonline.com/error?code=7000216
	AuthorizationFailed                = "AuthorizationFailed"
)

var RetryOptions = policy.RetryOptions{
	TryTimeout:  10 * time.Minute,
	ShouldRetry: shouldRetry,
}

// shouldRetry checks if the response is retriable.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// Retry if it gets an error because the error given to the function is not a non-retriable error.
		// https://github.com/Azure/azure-sdk-for-go/blob/cd497f0dad7a56807501606bb7e20cf710f863db/sdk/azcore/runtime/policy_retry.go#L151-L164
		return true
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return false
	}
	for _, sc := range autorest.StatusCodesForRetry {
		if resp.StatusCode == sc {
			return true
		}
	}

	// Check if the body contains the certain strings that can be retried.
	var b []byte
	_, err = resp.Body.Read(b)
	if err != nil {
		return true
	}
	body := string(b)
	return strings.Contains(body, ErrCodeInvalidClientSecretProvided) ||
		strings.Contains(body, ErrCodeMissingRequiredParameters) ||
		strings.Contains(body, AuthorizationFailed)
}
