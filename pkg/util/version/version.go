package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

const (
	OpenShiftVersion        = "4.3.8"
	OpenShiftPullSpecFormat = "%s.azurecr.io/openshift-release-dev/ocp-release@sha256:a414f6308db72f88e9d2e95018f0cc4db71c6b12b2ec0f44587488f0a16efc42"
)

func OpenShiftPullSpec(acrName string) string {
	return fmt.Sprintf(OpenShiftPullSpecFormat, acrName)
}
