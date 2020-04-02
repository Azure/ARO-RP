package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

const (
	OpenShiftVersion        = "4.3.9"
	OpenShiftPullSpecFormat = "%s.azurecr.io/openshift-release-dev/ocp-release@sha256:f0fada3c8216dc17affdd3375ff845b838ef9f3d67787d3d42a88dcd0f328eea"
)

func OpenShiftPullSpec(acrName string) string {
	return fmt.Sprintf(OpenShiftPullSpecFormat, acrName)
}
