package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
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
	UpgradeRP(context.Context) error
}

type deployer struct {
	log *logrus.Entry
	env env.Core

	globaldeployments      features.DeploymentsClient
	globalgroups           features.ResourceGroupsClient
	globalrecordsets       dns.RecordSetsClient
	globalaccounts         storage.AccountsClient
	deployments            features.DeploymentsClient
	features               features.Client
	groups                 features.ResourceGroupsClient
	userassignedidentities msi.UserAssignedIdentitiesClient
	providers              features.ProvidersClient
	publicipaddresses      network.PublicIPAddressesClient
	resourceskus           compute.ResourceSkusClient
	vmss                   compute.VirtualMachineScaleSetsClient
	vmssvms                compute.VirtualMachineScaleSetVMsClient
	zones                  dns.ZonesClient
	portalKeyvault         keyvault.Manager
	serviceKeyvault        keyvault.Manager

	config  *RPConfig
	version string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, env env.Core, config *RPConfig, version string) (Deployer, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	return &deployer{
		log: log,
		env: env,

		globaldeployments:      features.NewDeploymentsClient(env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalgroups:           features.NewResourceGroupsClient(env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:       dns.NewRecordSetsClient(env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		globalaccounts:         storage.NewAccountsClient(env.Environment(), *config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:            features.NewDeploymentsClient(env.Environment(), config.SubscriptionID, authorizer),
		features:               features.NewClient(env.Environment(), config.SubscriptionID, authorizer),
		groups:                 features.NewResourceGroupsClient(env.Environment(), config.SubscriptionID, authorizer),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(env.Environment(), config.SubscriptionID, authorizer),
		providers:              features.NewProvidersClient(env.Environment(), config.SubscriptionID, authorizer),
		resourceskus:           compute.NewResourceSkusClient(env.Environment(), config.SubscriptionID, authorizer),
		publicipaddresses:      network.NewPublicIPAddressesClient(env.Environment(), config.SubscriptionID, authorizer),
		vmss:                   compute.NewVirtualMachineScaleSetsClient(env.Environment(), config.SubscriptionID, authorizer),
		vmssvms:                compute.NewVirtualMachineScaleSetVMsClient(env.Environment(), config.SubscriptionID, authorizer),
		zones:                  dns.NewZonesClient(env.Environment(), config.SubscriptionID, authorizer),
		portalKeyvault:         keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+"-por."+env.Environment().KeyVaultDNSSuffix+"/"),
		serviceKeyvault:        keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+"-svc."+env.Environment().KeyVaultDNSSuffix+"/"),

		config:  config,
		version: version,
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
		case "portalAccessGroupIds", "portalElevatedGroupIds":
			v = strings.Join(v.([]string), ",")
		}

		parameters.Parameters[p] = &arm.ParametersParameter{
			Value: v,
		}
	}

	return parameters
}

func (d *deployer) encryptionAtHostSupported(ctx context.Context) (bool, error) {
	skus, err := d.resourceskus.List(ctx, "")
	if err != nil {
		return false, err
	}

	for _, sku := range skus {
		if !strings.EqualFold((*sku.Locations)[0], d.config.Location) ||
			*sku.Name != "Standard_D2s_v3" {
			continue
		}

		for _, cap := range *sku.Capabilities {
			if *cap.Name == "EncryptionAtHostSupported" &&
				*cap.Value == "True" {
				return true, nil
			}
		}
	}

	return false, nil
}
