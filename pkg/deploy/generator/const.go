package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Deployment constants
const (
	KeyVaultTagName          = "vault"
	ClustersKeyVaultTagValue = "clusters"
	ServiceKeyVaultTagValue  = "service"
	kvClusterSuffix          = "-cls"
	kvServiceSuffix          = "-svc"
)

// Template file constants
const (
	FileRPProductionPredeploy           = "rp-production-predeploy.json"
	FileRPProduction                    = "rp-production.json"
	FileRPProductionManagedIdentity     = "rp-production-managed-identity.json"
	fileRPProductionParameters          = "rp-production-parameters.json"
	fileRPProductionPredeployParameters = "rp-production-predeploy-parameters.json"

	fileEnvDevelopment         = "env-development.json"
	fileDatabaseDevelopment    = "databases-development.json"
	fileRPDevelopmentPredeploy = "rp-development-predeploy.json"
	fileRPDevelopment          = "rp-development.json"
)
