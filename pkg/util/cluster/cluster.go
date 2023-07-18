package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/jongio/azidext/go/azidext"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	mgmtredhatopenshift20220904 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2022-09-04/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy/assets"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	keyvaultclient "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	redhatopenshift20200430 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2020-04-30/redhatopenshift"
	redhatopenshift20210901preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2021-09-01-preview/redhatopenshift"
	redhatopenshift20220401 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-04-01/redhatopenshift"
	redhatopenshift20220904 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-09-04/redhatopenshift"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Cluster struct {
	log          *logrus.Entry
	env          env.Core
	ci           bool
	ciParentVnet string

	spGraphClient                     *msgraph.GraphServiceClient
	deployments                       features.DeploymentsClient
	groups                            features.ResourceGroupsClient
	openshiftclustersv20200430        redhatopenshift20200430.OpenShiftClustersClient
	openshiftclustersv20210901preview redhatopenshift20210901preview.OpenShiftClustersClient
	openshiftclustersv20220401        redhatopenshift20220401.OpenShiftClustersClient
	openshiftclustersv20220904        redhatopenshift20220904.OpenShiftClustersClient
	securitygroups                    network.SecurityGroupsClient
	subnets                           network.SubnetsClient
	routetables                       network.RouteTablesClient
	roleassignments                   authorization.RoleAssignmentsClient
	peerings                          network.VirtualNetworkPeeringsClient
	ciParentVnetPeerings              network.VirtualNetworkPeeringsClient
	vaultsClient                      keyvaultclient.VaultsClient
}

type errors []error

func (errs errors) Error() string {
	var sb strings.Builder

	for _, err := range errs {
		sb.WriteString(err.Error())
		sb.WriteByte('\n')
	}

	return sb.String()
}

func New(log *logrus.Entry, environment env.Core, ci bool) (*Cluster, error) {
	if env.IsLocalDevelopmentMode() {
		if err := env.ValidateVars("AZURE_FP_CLIENT_ID"); err != nil {
			return nil, err
		}
	}

	options := environment.Environment().EnvironmentCredentialOptions()
	spTokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		return nil, err
	}

	spGraphClient, err := environment.Environment().NewGraphServiceClient(spTokenCredential)
	if err != nil {
		return nil, err
	}

	scopes := []string{environment.Environment().ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(spTokenCredential, scopes)

	c := &Cluster{
		log: log,
		env: environment,
		ci:  ci,

		spGraphClient:                     spGraphClient,
		deployments:                       features.NewDeploymentsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		groups:                            features.NewResourceGroupsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		openshiftclustersv20200430:        redhatopenshift20200430.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		openshiftclustersv20210901preview: redhatopenshift20210901preview.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		openshiftclustersv20220401:        redhatopenshift20220401.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		openshiftclustersv20220904:        redhatopenshift20220904.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		securitygroups:                    network.NewSecurityGroupsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		subnets:                           network.NewSubnetsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		routetables:                       network.NewRouteTablesClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		roleassignments:                   authorization.NewRoleAssignmentsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		peerings:                          network.NewVirtualNetworkPeeringsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		vaultsClient:                      keyvaultclient.NewVaultsClient(environment.Environment(), environment.SubscriptionID(), authorizer),
	}

	if ci && env.IsLocalDevelopmentMode() {
		// Only peer if CI=true and RP_MODE=development
		c.ciParentVnet = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vpn-vnet", c.env.SubscriptionID(), c.env.ResourceGroup())

		r, err := azure.ParseResourceID(c.ciParentVnet)
		if err != nil {
			return nil, err
		}

		c.ciParentVnetPeerings = network.NewVirtualNetworkPeeringsClient(environment.Environment(), r.SubscriptionID, authorizer)
	}

	return c, nil
}

func (c *Cluster) Create(ctx context.Context, vnetResourceGroup, clusterName string, osClusterVersion string) error {
	clusterGet, err := c.openshiftclustersv20220904.Get(ctx, vnetResourceGroup, clusterName)
	if err == nil {
		if clusterGet.ProvisioningState == mgmtredhatopenshift20220904.Failed {
			return fmt.Errorf("cluster exists and is in failed provisioning state, please delete and retry")
		}
		c.log.Print("cluster already exists, skipping create")
		return nil
	}

	fpSPID := os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")

	if fpSPID == "" {
		return fmt.Errorf("fp service principal id is not found")
	}

	c.log.Infof("creating AAD application")
	appID, appSecret, err := c.createApplication(ctx, "aro-"+clusterName)
	if err != nil {
		return err
	}

	spID, err := c.createServicePrincipal(ctx, appID)
	if err != nil {
		return err
	}

	visibility := api.VisibilityPublic

	if os.Getenv("PRIVATE_CLUSTER") != "" || os.Getenv("NO_INTERNET") != "" {
		visibility = api.VisibilityPrivate
	}

	if c.ci {
		c.log.Infof("creating resource group")
		_, err = c.groups.CreateOrUpdate(ctx, vnetResourceGroup, mgmtfeatures.ResourceGroup{
			Location: to.StringPtr(c.env.Location()),
		})
		if err != nil {
			return err
		}
	}

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileClusterPredeploy)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
	if err != nil {
		return err
	}

	addressPrefix, masterSubnet, workerSubnet := c.generateSubnets()
	if err != nil {
		return err
	}

	var kvName string
	if len(vnetResourceGroup) > 10 {
		// keyvault names need to have a maximum length of 24,
		// so we need to cut off some chars if the resource group name is too long
		kvName = vnetResourceGroup[:10] + generator.SharedKeyVaultNameSuffix
	} else {
		kvName = vnetResourceGroup + generator.SharedKeyVaultNameSuffix
	}

	if c.ci {
		// name is limited to 24 characters, but must be globally unique, so we generate one and try if it is available
		kvName = "kv-" + uuid.DefaultGenerator.Generate()[:21]
		result, err := c.vaultsClient.CheckNameAvailability(ctx, mgmtkeyvault.VaultCheckNameAvailabilityParameters{Name: &kvName, Type: to.StringPtr("Microsoft.KeyVault/vaults")})
		if err != nil {
			return err
		}

		if result.NameAvailable != nil && !*result.NameAvailable {
			return fmt.Errorf("could not generate unique key vault name: %v", result.Reason)
		}
	}

	parameters := map[string]*arm.ParametersParameter{
		"clusterName":               {Value: clusterName},
		"ci":                        {Value: c.ci},
		"clusterServicePrincipalId": {Value: spID},
		"fpServicePrincipalId":      {Value: fpSPID},
		"vnetAddressPrefix":         {Value: addressPrefix},
		"masterAddressPrefix":       {Value: masterSubnet},
		"workerAddressPrefix":       {Value: workerSubnet},
		"kvName":                    {Value: kvName},
	}

	// TODO: ick
	if os.Getenv("NO_INTERNET") != "" {
		parameters["routes"] = &arm.ParametersParameter{
			Value: []mgmtnetwork.Route{
				{
					RoutePropertiesFormat: &mgmtnetwork.RoutePropertiesFormat{
						AddressPrefix: to.StringPtr("0.0.0.0/0"),
						NextHopType:   mgmtnetwork.RouteNextHopTypeNone,
					},
					Name: to.StringPtr("blackhole"),
				},
			},
		}
	}

	armctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	c.log.Info("predeploying ARM template")
	err = c.deployments.CreateOrUpdateAndWait(armctx, vnetResourceGroup, clusterName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Parameters: parameters,
			Mode:       mgmtfeatures.Incremental,
		},
	})
	if err != nil {
		return err
	}

	diskEncryptionSetID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/diskEncryptionSets/%s%s",
		c.env.SubscriptionID(),
		vnetResourceGroup,
		vnetResourceGroup,
		generator.SharedDiskEncryptionSetNameSuffix,
	)

	c.log.Info("creating role assignments")
	for _, scope := range []struct{ resource, role string }{
		{"/subscriptions/" + c.env.SubscriptionID() + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/dev-vnet", rbac.RoleNetworkContributor},
		{"/subscriptions/" + c.env.SubscriptionID() + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/routeTables/" + clusterName + "-rt", rbac.RoleNetworkContributor},
		{diskEncryptionSetID, rbac.RoleReader},
	} {
		for _, principalID := range []string{spID, fpSPID} {
			for i := 0; i < 5; i++ {
				_, err = c.roleassignments.Create(
					ctx,
					scope.resource,
					uuid.DefaultGenerator.Generate(),
					mgmtauthorization.RoleAssignmentCreateParameters{
						RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
							RoleDefinitionID: to.StringPtr("/subscriptions/" + c.env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/" + scope.role),
							PrincipalID:      &principalID,
							PrincipalType:    mgmtauthorization.ServicePrincipal,
						},
					},
				)

				// Ignore if the role assignment already exists
				if detailedError, ok := err.(autorest.DetailedError); ok {
					if detailedError.StatusCode == http.StatusConflict {
						err = nil
					}
				}

				// TODO: tighten this error check
				if err != nil && i < 4 {
					// Sometimes we see HashConflictOnDifferentRoleAssignmentIds.
					// Retry a few times.
					c.log.Print(err)
					continue
				}
				if err != nil {
					return err
				}

				break
			}
		}
	}

	c.log.Info("creating cluster")
	err = c.createCluster(ctx, vnetResourceGroup, clusterName, appID, appSecret, diskEncryptionSetID, visibility, osClusterVersion)

	if err != nil {
		return err
	}

	if c.ci {
		c.log.Info("fixing up NSGs")
		err = c.fixupNSGs(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			return err
		}

		if env.IsLocalDevelopmentMode() {
			c.log.Info("peering subnets to CI infra")
			err = c.peerSubnetsToCI(ctx, vnetResourceGroup, clusterName)
			if err != nil {
				return err
			}
		}
	}

	c.log.Info("done")
	return nil
}

func (c *Cluster) generateSubnets() (vnetPrefix string, masterSubnet string, workerSubnet string) {
	// pick a random 23 in range [10.3.0.0, 10.127.255.0]
	// 10.0.0.0/16 is used by dev-vnet to host CI
	// 10.1.0.0/24 is used by rp-vnet to host Proxy VM
	// 10.2.0.0/24 is used by dev-vpn-vnet to host VirtualNetworkGateway
	var x, y int
	rand.Seed(time.Now().UnixNano())
	// Local Dev clusters are limited to /16 dev-vnet
	if !c.ci {
		x, y = 0, 2*rand.Intn(128)
	} else {
		x, y = rand.Intn((124))+3, 2*rand.Intn(128)
	}
	vnetPrefix = fmt.Sprintf("10.%d.%d.0/23", x, y)
	masterSubnet = fmt.Sprintf("10.%d.%d.0/24", x, y)
	workerSubnet = fmt.Sprintf("10.%d.%d.0/24", x, y+1)
	return
}

func (c *Cluster) Delete(ctx context.Context, vnetResourceGroup, clusterName string) error {
	var errs errors

	oc, err := c.openshiftclustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
	if err == nil {
		err = c.deleteRoleAssignments(ctx, vnetResourceGroup, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}

		err = c.deleteApplication(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Print("deleting cluster")
		err = c.openshiftclustersv20200430.DeleteAndWait(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if c.ci {
		_, err = c.groups.Get(ctx, vnetResourceGroup)
		if err == nil {
			c.log.Print("deleting resource group")
			err = c.groups.DeleteAndWait(ctx, vnetResourceGroup)
			if err != nil {
				errs = append(errs, err)
			}
		}
		// Only delete peering if CI=true and RP_MODE=development
		if env.IsLocalDevelopmentMode() {
			r, err := azure.ParseResourceID(c.ciParentVnet)
			if err == nil {
				err = c.ciParentVnetPeerings.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, vnetResourceGroup+"-peer")
			}
			if err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		// Deleting the deployment does not clean up the associated resources
		c.log.Info("deleting deployment")
		err = c.deployments.DeleteAndWait(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting master/worker subnets")
		err = c.subnets.DeleteAndWait(ctx, vnetResourceGroup, "dev-vnet", clusterName+"-master")
		if err != nil {
			errs = append(errs, err)
		}

		err = c.subnets.DeleteAndWait(ctx, vnetResourceGroup, "dev-vnet", clusterName+"-worker")
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting route table")
		err = c.routetables.DeleteAndWait(ctx, vnetResourceGroup, clusterName+"-rt")
		if err != nil {
			errs = append(errs, err)
		}
	}

	c.log.Info("done")

	if errs != nil {
		return errs // https://golang.org/doc/faq#nil_error
	}

	return nil
}

// createCluster created new clusters, based on where it is running.
// development - using preview api
// production - using stable GA api
func (c *Cluster) createCluster(ctx context.Context, vnetResourceGroup, clusterName, clientID, clientSecret, diskEncryptionSetID string, visibility api.Visibility, osClusterVersion string) error {
	// using internal representation for "singe source" of options
	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain:               strings.ToLower(clusterName),
				ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.env.SubscriptionID(), "aro-"+clusterName),
				FipsValidatedModules: api.FipsValidatedModulesEnabled,
				Version:              osClusterVersion,
			},
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID:     clientID,
				ClientSecret: api.SecureString(clientSecret),
			},
			NetworkProfile: api.NetworkProfile{
				PodCIDR:                "10.128.0.0/14",
				ServiceCIDR:            "172.30.0.0/16",
				SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
			},
			MasterProfile: api.MasterProfile{
				VMSize:              api.VMSizeStandardD8sV3,
				SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.env.SubscriptionID(), vnetResourceGroup, clusterName),
				EncryptionAtHost:    api.EncryptionAtHostEnabled,
				DiskEncryptionSetID: diskEncryptionSetID,
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:                "worker",
					VMSize:              api.VMSizeStandardD4sV3,
					DiskSizeGB:          128,
					SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", c.env.SubscriptionID(), vnetResourceGroup, clusterName),
					Count:               3,
					EncryptionAtHost:    api.EncryptionAtHostEnabled,
					DiskEncryptionSetID: diskEncryptionSetID,
				},
			},
			APIServerProfile: api.APIServerProfile{
				Visibility: visibility,
			},
			IngressProfiles: []api.IngressProfile{
				{
					Name:       "default",
					Visibility: visibility,
				},
			},
		},
		Location: c.env.Location(),
	}

	if c.env.IsLocalDevelopmentMode() {
		err := c.registerSubscription(ctx)
		if err != nil {
			return err
		}

		err = c.insertDefaultVersionIntoCosmosdb(ctx)
		if err != nil {
			return err
		}

		oc.Properties.WorkerProfiles[0].VMSize = api.VMSizeStandardD2sV3
	}

	ext := api.APIs[v20220904.APIVersion].OpenShiftClusterConverter.ToExternal(&oc)
	data, err := json.Marshal(ext)
	if err != nil {
		return err
	}

	ocExt := mgmtredhatopenshift20220904.OpenShiftCluster{}
	err = json.Unmarshal(data, &ocExt)
	if err != nil {
		return err
	}

	return c.openshiftclustersv20220904.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, ocExt)
}

func (c *Cluster) registerSubscription(ctx context.Context) error {
	b, err := json.Marshal(&api.Subscription{
		State: api.SubscriptionStateRegistered,
		Properties: &api.SubscriptionProperties{
			TenantID: c.env.TenantID(),
			RegisteredFeatures: []api.RegisteredFeatureProfile{
				{
					Name:  "Microsoft.RedHatOpenShift/RedHatEngineering",
					State: "Registered",
				},
			},
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, "https://localhost:8443/subscriptions/"+c.env.SubscriptionID()+"?api-version=2.0", bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := cli.Do(req)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func (c *Cluster) insertDefaultVersionIntoCosmosdb(ctx context.Context) error {
	defaultVersion := version.DefaultInstallStream
	b, err := json.Marshal(&api.OpenShiftVersion{
		Properties: api.OpenShiftVersionProperties{
			Version:           defaultVersion.Version.String(),
			OpenShiftPullspec: defaultVersion.PullSpec,
			// HACK: we hardcode this to the latest installer image in arointsvc
			// if it is not overridden with ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC or LiveConfig
			InstallerPullspec: fmt.Sprintf("arointsvc.azurecr.io/aro-installer:release-%s", version.DefaultInstallStream.Version.MinorVersion()),
			Enabled:           true,
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, "https://localhost:8443/admin/versions", bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := cli.Do(req)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func (c *Cluster) fixupNSGs(ctx context.Context, vnetResourceGroup, clusterName string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// very occasionally c.securitygroups.List returns an empty list in
	// production.  No idea why.  Let's try retrying it...
	var nsgs []mgmtnetwork.SecurityGroup
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var err error
		nsgs, err = c.securitygroups.List(ctx, "aro-"+clusterName)
		return len(nsgs) > 0, err
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	for _, subnetName := range []string{clusterName + "-master", clusterName + "-worker"} {
		subnet, err := c.subnets.Get(ctx, vnetResourceGroup, "dev-vnet", subnetName, "")
		if err != nil {
			return err
		}

		subnet.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: nsgs[0].ID,
		}

		err = c.subnets.CreateOrUpdateAndWait(ctx, vnetResourceGroup, "dev-vnet", subnetName, subnet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) deleteRoleAssignments(ctx context.Context, vnetResourceGroup, appID string) error {
	spObjID, err := utilgraph.GetServicePrincipalIDByAppID(ctx, c.spGraphClient, appID)
	if err != nil {
		return err
	}
	if spObjID == nil {
		return nil
	}

	roleAssignments, err := c.roleassignments.ListForResourceGroup(ctx, vnetResourceGroup, fmt.Sprintf("principalId eq '%s'", *spObjID))
	if err != nil {
		return err
	}

	for _, roleAssignment := range roleAssignments {
		if strings.HasPrefix(
			strings.ToLower(*roleAssignment.Scope),
			strings.ToLower("/subscriptions/"+c.env.SubscriptionID()+"/resourceGroups/"+vnetResourceGroup),
		) {
			// Don't delete inherited role assignments, only those resource group level or below
			c.log.Infof("deleting role assignment %s", *roleAssignment.Name)
			_, err = c.roleassignments.Delete(ctx, *roleAssignment.Scope, *roleAssignment.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cluster) peerSubnetsToCI(ctx context.Context, vnetResourceGroup, clusterName string) error {
	cluster := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet", c.env.SubscriptionID(), vnetResourceGroup)

	r, err := azure.ParseResourceID(c.ciParentVnet)
	if err != nil {
		return err
	}

	clusterProp := &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &mgmtnetwork.SubResource{
			ID: &c.ciParentVnet,
		},
		AllowVirtualNetworkAccess: to.BoolPtr(true),
		AllowForwardedTraffic:     to.BoolPtr(true),
		UseRemoteGateways:         to.BoolPtr(true),
	}
	rpProp := &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &mgmtnetwork.SubResource{
			ID: &cluster,
		},
		AllowVirtualNetworkAccess: to.BoolPtr(true),
		AllowForwardedTraffic:     to.BoolPtr(true),
		AllowGatewayTransit:       to.BoolPtr(true),
	}

	err = c.peerings.CreateOrUpdateAndWait(ctx, vnetResourceGroup, "dev-vnet", r.ResourceGroup+"-peer", mgmtnetwork.VirtualNetworkPeering{VirtualNetworkPeeringPropertiesFormat: clusterProp})
	if err != nil {
		return err
	}

	err = c.ciParentVnetPeerings.CreateOrUpdateAndWait(ctx, r.ResourceGroup, r.ResourceName, vnetResourceGroup+"-peer", mgmtnetwork.VirtualNetworkPeering{VirtualNetworkPeeringPropertiesFormat: rpProp})
	if err != nil {
		return err
	}

	return err
}
