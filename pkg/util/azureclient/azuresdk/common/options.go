package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
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
	TryTimeout:    10 * time.Minute,
	MaxRetries:    3,                // 4 total attempts
	RetryDelay:    15 * time.Second, // ARM conflicts need time to clear; also governs 500/503 without Retry-After
	MaxRetryDelay: 60 * time.Second,
	ShouldRetry:   shouldRetry,
}

// shouldRetry retries on HTTP infrastructure signals only. Body-based semantic
// detection is handled at the call-site level via arm.Retryable() and
// IsRetryableError, after the SDK has fully deserialized the response.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
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
	// 409 with Retry-After header indicates a transient ARM conflict.
	return resp.StatusCode == http.StatusConflict && resp.Header.Get("Retry-After") != ""
}
