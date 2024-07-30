package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
)

const (
	EnvDatabaseName        = "DATABASE_NAME"
	EnvDatabaseAccountName = "DATABASE_ACCOUNT_NAME"
)

// Fetch the database account name from the environment.
func DBAccountName() (string, error) {
	if err := ValidateVars(EnvDatabaseAccountName); err != nil {
		return "", err
	}

	return os.Getenv(EnvDatabaseAccountName), nil
}

func DBName(c Core) (string, error) {
	if !c.IsLocalDevelopmentMode() {
		return "ARO", nil
	}

	if err := ValidateVars(EnvDatabaseName); err != nil {
		return "", fmt.Errorf("%v (development mode)", err.Error())
	}

	return os.Getenv(EnvDatabaseName), nil
}
