package azureclaim

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

type AzureClaim struct {
	Roles    []string `json:"roles,omitempty"`
	TenantID string   `json:"tid,omitempty"`
}

func (*AzureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}
