package config

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type MUODeploymentConfig struct {
	EnableConnected              bool
	OCMBaseURL                   string
	Pullspec                     string
	SupportsPodSecurityAdmission bool
}
