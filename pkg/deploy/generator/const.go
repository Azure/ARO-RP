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

	fileMiwiDevelopment = "rp-development-miwi.json"
	fileCIDevelopment   = "ci-development.json"

	// Tag constants
	tagKeyExemptPublicBlob   = "Az.Sec.AnonymousBlobAccessEnforcement::Skip"
	tagValueExemptPublicBlob = "PublicRelease"

	// CosmosDB Trigger Functions
	renewLeaseTriggerFunction = `function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				body["leaseExpires"] = Math.floor(date.getTime() / 1000) + 60;
				request.setBody(body);
			}`

	retryLaterTriggerFunction = `function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				body["leaseExpires"] = Math.floor(date.getTime() / 1000) + 600;
				request.setBody(body);
			}`

	setCreationBillingTimeStampTriggerFunction = `function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				var now = Math.floor(date.getTime() / 1000);
				var billingBody = body["billing"];
				if (!billingBody["creationTime"]) {
					billingBody["creationTime"] = now;
				}
				request.setBody(body);
			}`

	setDeletionBillingTimeStampTriggerFunction = `function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				var now = Math.floor(date.getTime() / 1000);
				var billingBody = body["billing"];
				if (!billingBody["deletionTime"]) {
					billingBody["deletionTime"] = now;
				}
				request.setBody(body);
			}`
)
