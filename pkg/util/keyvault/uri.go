package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

func URI(instancemetadata instancemetadata.InstanceMetadata, suffix, keyVaultPrefix string) string {
	return fmt.Sprintf("https://%s%s.%s/", keyVaultPrefix, suffix, instancemetadata.Environment().KeyVaultDNSSuffix)
}
