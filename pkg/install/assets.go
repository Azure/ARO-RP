//go:generate go run ../../hack/assets

package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/openshift/installer/data"
)

func init() {
	data.Assets = Assets
}
