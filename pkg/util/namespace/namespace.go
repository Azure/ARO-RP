package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

// IsOpenShift returns true if input name coresponds to openshift management
// namespaces pattern.
func IsOpenShift(ns string) bool {
	// openshift-operators for now is considered non-openshift namespace
	// because it is in user controll
	if ns == "openshift-operators" {
		return false
	}
	return ns == "" ||
		ns == "default" ||
		ns == "openshift" ||
		strings.HasPrefix(ns, "kube-") ||
		strings.HasPrefix(ns, "openshift-")
}
