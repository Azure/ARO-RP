package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/dgrijalva/jwt-go"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context, *api.OpenShiftCluster) error
}

func NewOpenShiftClusterDynamicValidator(env env.Interface) OpenShiftClusterDynamicValidator {
	return &openShiftClusterDynamicValidator{
		env: env,
	}
}

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type openShiftClusterDynamicValidator struct {
	env env.Interface
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context, oc *api.OpenShiftCluster) error {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return err
	}

	// TODO: pre-check that the cluster domain doesn't already exist

	spAuthorizer, err := dv.validateServicePrincipalProfile(oc)
	if err != nil {
		return err
	}

	err = dv.validateServicePrincipalRole(oc)
	if err != nil {
		return err
	}

	spPermissions := authorization.NewPermissionsClient(r.SubscriptionID, spAuthorizer)
	err = dv.validateVnetPermissions(ctx, oc, spPermissions, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	fpAuthorizer, err := dv.env.FPAuthorizer(oc.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	fpPermissions := authorization.NewPermissionsClient(r.SubscriptionID, fpAuthorizer)
	err = dv.validateVnetPermissions(ctx, oc, fpPermissions, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	subnetManager := subnet.NewManager(r.SubscriptionID, spAuthorizer)
	return dv.validateSubnets(ctx, subnetManager, oc)
}

func (dv *openShiftClusterDynamicValidator) validateServicePrincipalProfile(oc *api.OpenShiftCluster) (autorest.Authorizer, error) {
	spp := &oc.Properties.ServicePrincipalProfile
	conf := auth.NewClientCredentialsConfig(spp.ClientID, string(spp.ClientSecret), spp.TenantID)

	token, err := conf.ServicePrincipalToken()
	if err != nil {
		return nil, err
	}

	wait.PollImmediate(time.Second, 30*time.Second, func() (done bool, err error) {
		err = token.EnsureFresh()
		return err == nil, nil
	})
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal credentials are invalid.")
	}

	return autorest.NewBearerAuthorizer(token), nil
}

func (dv *openShiftClusterDynamicValidator) validateServicePrincipalRole(oc *api.OpenShiftCluster) error {
	spp := &oc.Properties.ServicePrincipalProfile
	conf := auth.NewClientCredentialsConfig(spp.ClientID, string(spp.ClientSecret), spp.TenantID)
	conf.Resource = azure.PublicCloud.GraphEndpoint

	token, err := conf.ServicePrincipalToken()
	if err != nil {
		return err
	}

	wait.PollImmediate(time.Second, 30*time.Second, func() (done bool, err error) {
		err = token.EnsureFresh()
		return err == nil, nil
	})
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateVnetPermissions(ctx context.Context, oc *api.OpenShiftCluster, client authorization.PermissionsClient, code, typ string) error {
	vnetID, _, err := subnet.Split(oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	permissions, err := client.ListForResource(ctx, r.ResourceGroup, r.Provider, r.ResourceType, "", r.ResourceName)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnetID)
	}
	if err != nil {
		return err
	}

	for _, action := range []string{
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	} {
		ok, err := utilpermissions.CanDoAction(permissions, action)
		if err != nil {
			return err
		}
		if !ok {
			return api.NewCloudError(http.StatusBadRequest, code, "", "The "+typ+" does not have Contributor permission on vnet '%s'.", vnetID)
		}
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateSubnets(ctx context.Context, subnetManager subnet.Manager, oc *api.OpenShiftCluster) error {
	master, err := dv.validateSubnet(ctx, subnetManager, oc, "properties.masterProfile.subnetId", "master", oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	worker, err := dv.validateSubnet(ctx, subnetManager, oc, `properties.workerProfiles["worker"].subnetId`, "worker", oc.Properties.WorkerProfiles[0].SubnetID)
	if err != nil {
		return err
	}

	_, pod, err := net.ParseCIDR(oc.Properties.NetworkProfile.PodCIDR)
	if err != nil {
		return err
	}

	_, service, err := net.ParseCIDR(oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	err = cidr.VerifyNoOverlap([]*net.IPNet{master, worker, pod, service}, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: '%s'.", err)
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateSubnet(ctx context.Context, subnetManager subnet.Manager, oc *api.OpenShiftCluster, path, typ, subnetID string) (*net.IPNet, error) {
	s, err := subnetManager.Get(ctx, subnetID)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' could not be found.", subnetID)
	}
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(oc.Properties.MasterProfile.SubnetID, subnetID) {
		if !strings.EqualFold(*s.PrivateLinkServiceNetworkPolicies, "Disabled") {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' is invalid: must have privateLinkServiceNetworkPolicies disabled.", subnetID)
		}
	}

	var found bool
	if s.ServiceEndpoints != nil {
		for _, se := range *s.ServiceEndpoints {
			if strings.EqualFold(*se.Service, "Microsoft.ContainerRegistry") &&
				se.ProvisioningState == mgmtnetwork.Succeeded {
				found = true
				break
			}
		}
	}
	if !found {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.", subnetID)
	}

	if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		if s.SubnetPropertiesFormat != nil &&
			s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' is invalid: must not have a network security group attached.", subnetID)
		}

	} else {
		nsgID, err := subnet.NetworkSecurityGroupID(oc, *s.ID)
		if err != nil {
			return nil, err
		}

		if s.SubnetPropertiesFormat == nil ||
			s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
			!strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' is invalid: must have network security group '%s' attached.", subnetID, nsgID)
		}
	}

	_, net, err := net.ParseCIDR(*s.AddressPrefix)
	if err != nil {
		return nil, err
	}
	{
		ones, _ := net.Mask.Size()
		if ones > 27 {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided "+typ+" VM subnet '%s' is invalid: must be /27 or larger.", subnetID)
		}
	}

	return net, nil
}
