package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

// IsOpenShift returns true if ns is an openshift managed namespace.
func IsOpenShift(ns string) bool {
	return ns == "" ||
		ns == "default" ||
		ns == "openshift" ||
		strings.HasPrefix(ns, "kube-") ||
		strings.HasPrefix(ns, "openshift-")
}
