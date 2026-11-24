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

// IsManagedDomain returns true if domain is an ARO-managed domain.
// Managed domains are either the short form stored in cluster documents
// ("foo") or the FQDN form ("foo.$LOCATION.aroapp.io" /
// "foo.aroapp.azure.us"). Custom domains always contain dots and do not
// end with a managed suffix.
func IsManagedDomain(domain string) bool {
	if domain == "" ||
		strings.HasPrefix(domain, ".") ||
		strings.HasSuffix(domain, ".") {
		return false
	}

	// Short-form managed domain as stored in ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		return true
	}

	for _, suffix := range managedDomainSuffixes() {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}
	return false
}
