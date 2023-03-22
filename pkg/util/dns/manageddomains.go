package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "strings"

const (
	publicCloudManagedDomainSuffix = ".aroapp.io"
	govCloudManagedDomainSuffix    = ".aroapp.azure.us"
)

func managedDomainSuffixes() []string {
	return []string{
		publicCloudManagedDomainSuffix,
		govCloudManagedDomainSuffix,
	}
}

func IsManagedDomain(domain string) bool {
	for _, suffix := range managedDomainSuffixes() {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}
	return false
}
