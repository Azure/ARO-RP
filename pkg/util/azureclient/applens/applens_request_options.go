package applens

import "net/http"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type appLensRequestOptions interface {
	toHeader() http.Header
}
