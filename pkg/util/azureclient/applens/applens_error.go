package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

func newAppLensError(response *http.Response) error {
	bytesRead, err := runtime.Payload(response)
	if err != nil {
		return err
	}

	appLensError := azcore.ResponseError{
		StatusCode:  response.StatusCode,
		RawResponse: response,
	}

	// Attempt to extract Code from body
	var appLensErrorResponse appLensErrorResponse
	err = json.Unmarshal(bytesRead, &appLensErrorResponse)
	if err == nil {
		appLensError.ErrorCode = appLensErrorResponse.Code
	}

	return &appLensError
}

type appLensErrorResponse struct {
	Code string `json:"Code"`
}
