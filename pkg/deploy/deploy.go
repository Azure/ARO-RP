package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strings"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/vmsscleaner"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
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
	globalrecordsets       dns.RecordSetsClient
	globalaccounts         storage.AccountsClient
	deployments            features.DeploymentsClient
	groups                 features.ResourceGroupsClient
	loadbalancers          network.LoadBalancersClient
	userassignedidentities msi.UserAssignedIdentitiesClient
	providers              features.ProvidersClient
	publicipaddresses      network.PublicIPAddressesClient
	resourceskus           compute.ResourceSkusClient
	roleassignments        authorization.RoleAssignmentsClient
	vmss                   compute.VirtualMachineScaleSetsClient
	vmssvms                compute.VirtualMachineScaleSetVMsClient
	zones                  dns.ZonesClient
	clusterKeyvault        keyvault.Manager
	dbtokenKeyvault        keyvault.Manager
	portalKeyvault         keyvault.Manager
	serviceKeyvault        keyvault.Manager

	config      *RPConfig
	version     string
	vmssCleaner vmsscleaner.Interface
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, _env env.Core, config *RPConfig, version string) (Deployer, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(_env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	vmssClient := compute.NewVirtualMachineScaleSetsClient(_env.Environment(), config.SubscriptionID, authorizer)

	return &deployer{
		log: log,
		env: _env,

		globaldeployments:      features.NewDeploymentsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalgroups:           features.NewResourceGroupsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:       dns.NewRecordSetsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalaccounts:         storage.NewAccountsClient(_env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:            features.NewDeploymentsClient(_env.Environment(), config.SubscriptionID, authorizer),
		groups:                 features.NewResourceGroupsClient(_env.Environment(), config.SubscriptionID, authorizer),
		loadbalancers:          network.NewLoadBalancersClient(_env.Environment(), config.SubscriptionID, authorizer),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(_env.Environment(), config.SubscriptionID, authorizer),
		providers:              features.NewProvidersClient(_env.Environment(), config.SubscriptionID, authorizer),
		roleassignments:        authorization.NewRoleAssignmentsClient(_env.Environment(), config.SubscriptionID, authorizer),
		resourceskus:           compute.NewResourceSkusClient(_env.Environment(), config.SubscriptionID, authorizer),
		publicipaddresses:      network.NewPublicIPAddressesClient(_env.Environment(), config.SubscriptionID, authorizer),
		vmss:                   vmssClient,
		vmssvms:                compute.NewVirtualMachineScaleSetVMsClient(_env.Environment(), config.SubscriptionID, authorizer),
		zones:                  dns.NewZonesClient(_env.Environment(), config.SubscriptionID, authorizer),
		clusterKeyvault:        keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+env.ClusterKeyvaultSuffix+"."+_env.Environment().KeyVaultDNSSuffix+"/"),
		dbtokenKeyvault:        keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+env.DBTokenKeyvaultSuffix+"."+_env.Environment().KeyVaultDNSSuffix+"/"),
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
	for i := 0; i < 3; i++ {
		d.log.Printf("deploying %s", deploymentName)
		err = d.deployments.CreateOrUpdateAndWait(ctx, rgName, deploymentName, deployment)
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "DeploymentFailed" &&
			i < 1 {
			// on new RP deployments, we get a spurious DeploymentFailed error
			// from the Microsoft.Insights/metricAlerts resources indicating
			// that rp-lb can't be found, even though it exists and the
			// resources correctly have a dependsOn stanza referring to it.
			// Retry once.
			d.log.Print(err)
			continue
		}
		if err != nil && *d.config.Configuration.VMSSCleanupEnabled {
			if retry := d.vmssCleaner.RemoveFailedNewScaleset(ctx, rgName, vmssName); retry {
				continue // Retry deployment after deleting failed VMSS.
			}
		}
		break
	}
	return err
}
