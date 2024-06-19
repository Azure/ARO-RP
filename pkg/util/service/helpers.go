package service

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
)

func DBName(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
		return "ARO", nil
	}

	if err := env.ValidateVars(DatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(DatabaseName), nil
}
