package errors

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// IsNotFoundError checks if the error is an error from azure SDK and 404 NotFound error.
func IsNotFoundError(err error) bool {
	var azErr *azcore.ResponseError
	return errors.As(err, &azErr) && azErr.StatusCode == http.StatusNotFound
}
