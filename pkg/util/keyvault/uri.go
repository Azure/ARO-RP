package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

func URI(instancemetadata instancemetadata.InstanceMetadata, suffix string) (string, error) {
	// TODO (Aldo): can't use env package due to import cycle errors
	for _, key := range []string{
		"KEYVAULT_PREFIX",
	} {
		if _, found := os.LookupEnv(key); !found {
			return "", fmt.Errorf("environment variable %q unset", key)
		}
	}

	return fmt.Sprintf("https://%s%s.%s/", os.Getenv("KEYVAULT_PREFIX"), suffix, instancemetadata.Environment().KeyVaultDNSSuffix), nil
}
