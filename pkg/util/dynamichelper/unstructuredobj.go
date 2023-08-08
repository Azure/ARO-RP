package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
)

func isKindUnstructured(groupKind string) bool {
	return strings.HasSuffix(groupKind, ".constraints.gatekeeper.sh")
}
