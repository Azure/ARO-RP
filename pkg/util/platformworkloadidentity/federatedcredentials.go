package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	base36Encoding             = 36
	maxFederatedCredNameLength = 120
	numberOfDelimiters         = 1
)

func GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId azure.Resource, serviceAccountName string) string {
	clusterResourceKey := fmt.Sprintf("%s-%s", serviceAccountName, clusterResourceId.ResourceName)
	name := fmt.Sprintf("%s-%s-%s", clusterResourceKey, serviceAccountName, identityResourceId.ResourceName)
	// the base-36 encoded string of a SHA-224 hash will typically be around 43 to 44 characters long.
	hash := sha256.Sum224([]byte(name))
	encodedName := (&big.Int{}).SetBytes(hash[:]).Text(base36Encoding)
	remainingChars := maxFederatedCredNameLength - len(encodedName) - numberOfDelimiters

	if remainingChars < len(clusterResourceKey) {
		return fmt.Sprintf("%s-%s", clusterResourceKey[:remainingChars], encodedName)
	}

	return fmt.Sprintf("%s-%s", clusterResourceKey, encodedName)
}
