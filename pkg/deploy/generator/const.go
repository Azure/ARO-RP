package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// Template file constants
	FileRPProductionManagedIdentity          = "rp-production-managed-identity.json"
	FileRPProductionPredeploy                = "rp-production-predeploy.json"
	FileRPProductionPredeployParameters      = "rp-production-predeploy-parameters.json"
	FileRPProduction                         = "rp-production.json"
	FileRPProductionGlobal                   = "rp-production-global.json"
	FileRPProductionGlobalACRReplication     = "rp-production-global-acr-replication.json"
	FileRPProductionGlobalSubscription       = "rp-production-global-subscription.json"
	FileRPProductionParameters               = "rp-production-parameters.json"
	FileRPProductionSubscription             = "rp-production-subscription.json"
	FileGatewayProductionManagedIdentity     = "gateway-production-managed-identity.json"
	FileGatewayProductionPredeploy           = "gateway-production-predeploy.json"
	FileGatewayProductionPredeployParameters = "gateway-production-predeploy-parameters.json"
	FileGatewayProduction                    = "gateway-production.json"
	FileGatewayProductionParameters          = "gateway-production-parameters.json"

	FileClusterPredeploy       = "cluster-development-predeploy.json"
	fileEnvDevelopment         = "env-development.json"
	fileDatabaseDevelopment    = "databases-development.json"
	fileRPDevelopmentPredeploy = "rp-development-predeploy.json"
	fileRPDevelopment          = "rp-development.json"

	fileOic = "rp-oic.json"

	// Tag constants
	tagKeyExemptPublicBlob   = "Az.Sec.AnonymousBlobAccessEnforcement::Skip"
	tagValueExemptPublicBlob = "PublicRelease"
)
