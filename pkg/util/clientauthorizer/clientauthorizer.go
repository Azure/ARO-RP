package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
)

type ClientAuthorizer interface {
	IsAuthorized(*http.Request) bool
	IsReady() bool
}
