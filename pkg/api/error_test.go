package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"
)

func TestWrapCloudErrorWithMessage(t *testing.T) {
	t.Run("uses defaults when wrapped cloud error is nil", func(t *testing.T) {
		err := WrapCloudErrorWithMessage(errors.New("outer failure"), nil)

		if err.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", err.StatusCode, http.StatusInternalServerError)
		}
		if err.Code != CloudErrorCodeInternalServerError {
			t.Fatalf("code = %q, want %q", err.Code, CloudErrorCodeInternalServerError)
		}
		if err.Target != "" {
			t.Fatalf("target = %q, want empty", err.Target)
		}
		if err.Message != "outer failure" {
			t.Fatalf("message = %q, want %q", err.Message, "outer failure")
		}
	})

	t.Run("preserves cloud error status and metadata", func(t *testing.T) {
		inner := NewCloudError(http.StatusConflict, CloudErrorCodeRequestNotAllowed, "controlPlaneInventory", "inner")
		inner.Details = []CloudErrorBody{
			{Code: "Nested", Message: "nested message", Target: "nestedTarget"},
		}

		err := WrapCloudErrorWithMessage(errors.New("outer failure with rollback context"), inner)

		if err.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want %d", err.StatusCode, http.StatusConflict)
		}
		if err.Code != CloudErrorCodeRequestNotAllowed {
			t.Fatalf("code = %q, want %q", err.Code, CloudErrorCodeRequestNotAllowed)
		}
		if err.Target != "controlPlaneInventory" {
			t.Fatalf("target = %q, want %q", err.Target, "controlPlaneInventory")
		}
		if err.Message != "outer failure with rollback context" {
			t.Fatalf("message = %q, want %q", err.Message, "outer failure with rollback context")
		}
		if len(err.Details) != 1 {
			t.Fatalf("details len = %d, want 1", len(err.Details))
		}

		inner.Details[0].Message = "mutated"
		if err.Details[0].Message != "nested message" {
			t.Fatalf("details were not copied, got %q", err.Details[0].Message)
		}
	})

	t.Run("keeps status but falls back on code and target when body missing", func(t *testing.T) {
		inner := &CloudError{
			StatusCode:     http.StatusTooManyRequests,
			CloudErrorBody: nil,
		}

		err := WrapCloudErrorWithMessage(errors.New("outer failure"), inner)

		if err.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want %d", err.StatusCode, http.StatusTooManyRequests)
		}
		if err.Code != CloudErrorCodeInternalServerError {
			t.Fatalf("code = %q, want %q", err.Code, CloudErrorCodeInternalServerError)
		}
		if err.Target != "" {
			t.Fatalf("target = %q, want empty", err.Target)
		}
		if err.Message != "outer failure" {
			t.Fatalf("message = %q, want %q", err.Message, "outer failure")
		}
	})
}
