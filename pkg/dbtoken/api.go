package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const Resource = "https://dbtoken.aro.azure.com/"

type tokenResponse struct {
	Token string `json:"token,omitempty"`
}
