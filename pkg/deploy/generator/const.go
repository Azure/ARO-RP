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
	FileRPStageGlobal                        = "rp-stage-global.json"
	FileRPProductionGlobalACRReplication     = "rp-production-global-acr-replication.json"
	FileRPProductionGlobalSubscription       = "rp-production-global-subscription.json"
	FileRPStageGlobalSubscription            = "rp-stage-global-subscription.json"
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

	fileMiwiDevelopment = "rp-development-miwi.json"
	fileCIDevelopment   = "ci-development.json"

	// Tag constants
	tagKeyExemptPublicBlob   = "Az.Sec.AnonymousBlobAccessEnforcement::Skip"
	tagValueExemptPublicBlob = "PublicRelease"

	renewLeaseTriggerFunction                  = "function trigger() {\n\t\t\t\tvar request = getContext().getRequest();\n\t\t\t\tvar body = request.getBody();\n\t\t\t\tvar date = new Date();\n\t\t\t\tbody[\"leaseExpires\"] = Math.floor(date.getTime() / 1000) + 60;\n\t\t\t\trequest.setBody(body);\n\t\t\t}"
	retryLaterTriggerFunction                  = "function trigger() {\n\t\t\t\tvar request = getContext().getRequest();\n\t\t\t\tvar body = request.getBody();\n\t\t\t\tvar date = new Date();\n\t\t\t\tbody[\"leaseExpires\"] = Math.floor(date.getTime() / 1000) + 600;\n\t\t\t\trequest.setBody(body);\n\t\t\t}"
	setCreationBillingTimeStampTriggerFunction = "function trigger() {\n\t\t\t\tvar request = getContext().getRequest();\n\t\t\t\tvar body = request.getBody();\n\t\t\t\tvar date = new Date();\n\t\t\t\tvar now = Math.floor(date.getTime() / 1000);\n\t\t\t\tvar billingBody = body[\"billing\"];\n\t\t\t\tif (!billingBody[\"creationTime\"]) {\n\t\t\t\t\tbillingBody[\"creationTime\"] = now;\n\t\t\t\t}\n\t\t\t\trequest.setBody(body);\n\t\t\t}"
	setDeletionBillingTimeStampTriggerFunction = "function trigger() {\n\t\t\t\tvar request = getContext().getRequest();\n\t\t\t\tvar body = request.getBody();\n\t\t\t\tvar date = new Date();\n\t\t\t\tvar now = Math.floor(date.getTime() / 1000);\n\t\t\t\tvar billingBody = body[\"billing\"];\n\t\t\t\tif (!billingBody[\"creationTime\"]) {\n\t\t\t\t\tbillingBody[\"creationTime\"] = now;\n\t\t\t\t}\n\t\t\t\trequest.setBody(body);\n\t\t\t}"
)
