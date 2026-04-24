package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"io"
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

	// 409 is not in autorest.StatusCodesForRetry; retry only when the Retry-After header is present.
	if resp.StatusCode == http.StatusConflict && resp.Header.Get("Retry-After") != "" {
		return true
	}

	// Check if the body contains the certain strings that can be retried.
	b, err := io.ReadAll(resp.Body)
	// Close the original body to release the HTTP connection, even on read error
	resp.Body.Close()
	if err != nil {
		return true
	}
	// resp.Body is a shared object (pointer), so we need to restore it
	// to original state so it can be read again by the SDK or for retries
	resp.Body = io.NopCloser(bytes.NewReader(b))

	body := string(b)
	return strings.Contains(body, ErrCodeInvalidClientSecretProvided) ||
		strings.Contains(body, ErrCodeMissingRequiredParameters) ||
		strings.Contains(body, AuthorizationFailed)
}
