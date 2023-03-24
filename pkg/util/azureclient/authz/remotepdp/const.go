package remotepdp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	modulename = "aro-pdpclient"
	// version is the semantic version of this module
	version = "0.0.1"
)

// AccessDecision can be: Allowed, NotAllowed, Denied.
type AccessDecision string

// AccessDecision possible returned values
const (
	Allowed    AccessDecision = "Allowed"
	NotAllowed AccessDecision = "NotAllowed"
	Denied     AccessDecision = "Denied"
)
