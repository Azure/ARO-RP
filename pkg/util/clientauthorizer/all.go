package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
)

type all struct{}

func NewAll() ClientAuthorizer {
	return &all{}
}

func (all) IsAuthorized(*http.Request) bool {
	return true
}

func (all) IsReady() bool {
	return true
}
