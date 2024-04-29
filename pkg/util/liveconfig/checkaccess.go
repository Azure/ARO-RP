package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (p *prod) UseCheckAccess(ctx context.Context) (bool, error) {
	// TODO: Replace with RP Live Service Config (KeyVault)
	checkAccess := p.cfg.GetString(useCheckAccess)
	if checkAccess == "enabled" {
		return true, nil
	}
	return false, nil
}
