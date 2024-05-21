package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"regexp"
)

// Regular expressions used to validate the format of resource names and IDs acceptable by API.
var (
	RxClusterID           = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]/providers/Microsoft\.RedHatOpenShift/openShiftClusters/[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)
	RxResourceGroupID     = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)
	RxSubnetID            = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]/providers/Microsoft\.Network/virtualNetworks/[-a-z0-9_.]{2,64}/subnets/[-a-z0-9_.]{2,80}$`)
	RxDiskEncryptionSetID = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]/providers/Microsoft\.Compute/diskEncryptionSets/[-a-z0-9_]{1,80}$`)
	RxDomainName          = regexp.MustCompile(`^` +
		`([a-z][-a-z0-9]{0,61}[a-z0-9])` +
		`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*` +
		`$`)
	RxDomainNameRFC1123 = regexp.MustCompile(`^` +
		`([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9])` +
		`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*` +
		`$`)
	// DO NOT MERGE - temporary changes to support installing versions with suffixes for testing
	RxInstallVersion = regexp.MustCompile(`^[4-9]{1}\.[0-9]{1,2}\.[0-9]{1,3}(-.*)?$`)
)
