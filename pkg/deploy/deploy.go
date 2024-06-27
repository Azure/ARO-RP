package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	sdkarm "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/vmsscleaner"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	sdkdns "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armdns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/msi"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

var _ Deployer = (*deployer)(nil)

type Deployer interface {
	PreDeploy(context.Context) error
	DeployRP(context.Context) error
	DeployGateway(context.Context) error
	UpgradeRP(context.Context) error
	UpgradeGateway(context.Context) error
	SaveVersion(context.Context) error
}

type deployer struct {
	log *logrus.Entry
	env env.Core

	globaldeployments      features.DeploymentsClient
	globalgroups           features.ResourceGroupsClient
	globalrecordsets       sdkdns.RecordSetsClient
	globalaccounts         storage.AccountsClient
	deployments            features.DeploymentsClient
	groups                 features.ResourceGroupsClient
	userassignedidentities msi.UserAssignedIdentitiesClient
	providers              features.ProvidersClient
	publicipaddresses      network.PublicIPAddressesClient
	resourceskus           compute.ResourceSkusClient
	roleassignments        authorization.RoleAssignmentsClient
	vmss                   compute.VirtualMachineScaleSetsClient
	vmssvms                compute.VirtualMachineScaleSetVMsClient
	zones                  sdkdns.ZonesClient
	clusterKeyvault        keyvault.Manager
	portalKeyvault         keyvault.Manager
	serviceKeyvault        keyvault.Manager

	config      *RPConfig
	version     string
	vmssCleaner vmsscleaner.Interface
}

// KnownDeploymentErrorType represents a type of error we encounter during an
// RP/gateway deployment that we know how to handle via automation.
type KnownDeploymentErrorType string

const (
	KnownDeploymentErrorTypeRPLBNotFound KnownDeploymentErrorType = "RPLBNotFound"
)

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, _env env.Core, config *RPConfig, version string, tokenCredential azcore.TokenCredential) (Deployer, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	scopes := []string{_env.Environment().ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)

	scopes = []string{_env.Environment().KeyVaultScope}
	kvAuthorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)

	vmssClient := compute.NewVirtualMachineScaleSetsClient(_env.Environment(), config.SubscriptionID, authorizer)
	options := sdkarm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: _env.Environment().Cloud,
		},
	}
	return &deployer{
		log: log,
		env: _env,

		globaldeployments:      features.NewDeploymentsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalgroups:           features.NewResourceGroupsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:       sdkdns.NewRecordSetsClient(*config.Configuration.GlobalSubscriptionID, authorizer, &options),
		globalaccounts:         storage.NewAccountsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:            features.NewDeploymentsClient(_env.Environment(), config.SubscriptionID, authorizer),
		groups:                 features.NewResourceGroupsClient(_env.Environment(), config.SubscriptionID, authorizer),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(_env.Environment(), config.SubscriptionID, authorizer),
		providers:              features.NewProvidersClient(_env.Environment(), config.SubscriptionID, authorizer),
		roleassignments:        authorization.NewRoleAssignmentsClient(_env.Environment(), config.SubscriptionID, authorizer),
		resourceskus:           compute.NewResourceSkusClient(_env.Environment(), config.SubscriptionID, authorizer),
		publicipaddresses:      network.NewPublicIPAddressesClient(_env.Environment(), config.SubscriptionID, authorizer),
		vmss:                   vmssClient,
		vmssvms:                compute.NewVirtualMachineScaleSetVMsClient(_env.Environment(), config.SubscriptionID, authorizer),
		zones:                  sdkdns.NewZonesClient(config.SubscriptionID, authorizer, options),
		clusterKeyvault:        keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+env.ClusterKeyvaultSuffix+"."+_env.Environment().KeyVaultDNSSuffix+"/"),
		portalKeyvault:         keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+env.PortalKeyvaultSuffix+"."+_env.Environment().KeyVaultDNSSuffix+"/"),
		serviceKeyvault:        keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+env.ServiceKeyvaultSuffix+"."+_env.Environment().KeyVaultDNSSuffix+"/"),

		config:      config,
		version:     version,
		vmssCleaner: vmsscleaner.New(log, vmssClient),
	}, nil
}

// getParameters returns an *arm.Parameters populated with parameter names and
// values.  The names are taken from the ps argument and the values are taken
// from d.config.Configuration.
func (d *deployer) getParameters(ps map[string]interface{}) *arm.Parameters {
	m := map[string]interface{}{}
	v := reflect.ValueOf(*d.config.Configuration)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsNil() {
			continue
		}

		m[strings.SplitN(v.Type().Field(i).Tag.Get("json"), ",", 2)[0]] = v.Field(i).Interface()
	}

	parameters := &arm.Parameters{
		Parameters: map[string]*arm.ParametersParameter{},
	}

	for p := range ps {
		// do not convert empty fields
		// makes default values templates work
		v, ok := m[p]
		if !ok {
			continue
		}

		switch p {
		case "gatewayDomains", "gatewayFeatures", "portalAccessGroupIds", "portalElevatedGroupIds", "rpFeatures":
			v = strings.Join(v.([]string), ",")
		}

		parameters.Parameters[p] = &arm.ParametersParameter{
			Value: v,
		}
	}

	return parameters
}

func (d *deployer) deploy(ctx context.Context, rgName, deploymentName, vmssName string, deployment mgmtfeatures.Deployment) (err error) {
	numAttempts := 3

	for i := 0; i < numAttempts; i++ {
		d.log.Printf("deploying %s", deploymentName)
		err = d.deployments.CreateOrUpdateAndWait(ctx, rgName, deploymentName, deployment)
		serviceErr, isServiceError := err.(*azure.ServiceError)

		// As long as this is not the final deployment attempt,
		// unconditionally log the error before inspecting it.
		if err != nil && i < numAttempts-1 {
			d.log.Print(err)
		}

		// Check for a known error that we know how to handle.
		if isServiceError {
			errorType, checkTypeErr := d.checkForKnownError(serviceErr, i)

			if checkTypeErr != nil {
				d.log.Printf("Encountered an error in checkForKnownError: %s", checkTypeErr)
			}

			// On new RP deployments, we get a spurious DeploymentFailed error
			// from the Microsoft.Insights/metricAlerts resources indicating
			// that rp-lb can't be found, even though it exists and the
			// resources correctly have a dependsOn stanza referring to it.
			// Retry once, and only if this error is encountered on the first
			// deployment attempt.
			if errorType == KnownDeploymentErrorTypeRPLBNotFound {
				d.log.Print("Deployment encountered known ResourceNotFound error for RP LB; retrying.")
				continue
			}
		}

		// For errors we don't know how to handle, delete the failed VMSS and retry the deployment.
		if err != nil && *d.config.Configuration.VMSSCleanupEnabled {
			if retry := d.vmssCleaner.RemoveFailedNewScaleset(ctx, rgName, vmssName); retry {
				continue
			}
		}
		break
	}
	return err
}

// checkForKnownError is a helper function that checks the errors nested within an Azure ServiceError
// for a known error and returns the corresponding KnownDeploymentErrorType if applicable.
func (d *deployer) checkForKnownError(serviceErr *azure.ServiceError, deployAttempt int) (KnownDeploymentErrorType, error) {
	if serviceErr.Code != "DeploymentFailed" || len(serviceErr.Details) == 0 {
		return "", nil
	}

	outerErr := azure.ServiceError{}
	jsonEncoded, err := json.Marshal(serviceErr.Details[0])

	if err != nil {
		return "", err
	}

	err = json.Unmarshal(jsonEncoded, &outerErr)

	if err != nil {
		return "", err
	}

	innerErr := azure.ServiceError{}
	err = json.Unmarshal([]byte(outerErr.Message), &innerErr)

	if err != nil {
		return "", err
	}

	isFirstAttempt := deployAttempt < 1
	isRPLBNotFound := innerErr.Code == "ResourceNotFound" && strings.Contains(innerErr.Message, "Microsoft.Network/loadBalancers/rp-lb")

	if isFirstAttempt && isRPLBNotFound {
		return KnownDeploymentErrorTypeRPLBNotFound, nil
	}

	return "", nil
}
