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
	FileRPProductionManagedIdentity      = "rp-production-managed-identity.json"
	FileRPProductionPredeploy            = "rp-production-predeploy.json"
	FileRPProductionPredeployParameters  = "rp-production-predeploy-parameters.json"
	FileRPProduction                     = "rp-production.json"
	FileRPProductionGlobal               = "rp-production-global.json"
	FileRPProductionGlobalACRReplication = "rp-production-global-acr-replication.json"
	FileRPProductionGlobalSubscription   = "rp-production-global-subscription.json"
	FileRPProductionParameters           = "rp-production-parameters.json"
	FileRPProductionSubscription         = "rp-production-subscription.json"

	FileClusterPredeploy       = "cluster-predeploy.json"
	fileEnvDevelopment         = "env-development.json"
	fileDatabaseDevelopment    = "databases-development.json"
	fileRPDevelopmentPredeploy = "rp-development-predeploy.json"
	fileRPDevelopment          = "rp-development.json"
)
