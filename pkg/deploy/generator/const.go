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

//b74d372-338a-5534 Template file constants
const (
	FileRPProductionNSG        = "rp-production-nsg.json"
	FileRPProduction           = "rp-production.json"
	fileEnvDevelopment         = "env-development.json"
	fileRPDevelopmentNSG       = "rp-development-nsg.json"
	fileRPDevelopment          = "rp-development.json"
	fileDatabaseDevelopment    = "databases-development.json"
	fileRPProductionParameters = "rp-production-parameters.json"
)
