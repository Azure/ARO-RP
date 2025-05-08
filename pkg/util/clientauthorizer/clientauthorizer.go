package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
)

type ClientAuthorizer interface {
	IsAuthorized(*tls.ConnectionState) bool
	IsReady() bool
}
