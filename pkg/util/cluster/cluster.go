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

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/api/v20210131preview"
	mgmtredhatopenshift20200430 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	mgmtredhatopenshift20210131preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	redhatopenshift20200430 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2020-04-30/redhatopenshift"
	redhatopenshift20210131preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

type Cluster struct {
	log          *logrus.Entry
	env          env.Core
	ci           bool
	ciParentVnet string

	deployments                       features.DeploymentsClient
	groups                            features.ResourceGroupsClient
	applications                      graphrbac.ApplicationsClient
	serviceprincipals                 graphrbac.ServicePrincipalClient
	openshiftclustersv20200430        redhatopenshift20200430.OpenShiftClustersClient
	openshiftclustersv20210131preview redhatopenshift20210131preview.OpenShiftClustersClient
	securitygroups                    network.SecurityGroupsClient
	subnets                           network.SubnetsClient
	routetables                       network.RouteTablesClient
	roleassignments                   authorization.RoleAssignmentsClient
	peerings                          network.VirtualNetworkPeeringsClient
	ciParentVnetPeerings              network.VirtualNetworkPeeringsClient
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

func New(log *logrus.Entry, env env.Core, ci bool) (*Cluster, error) {
	if env.IsLocalDevelopmentMode() {
		for _, key := range []string{
			"AZURE_FP_CLIENT_ID",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset", key)
			}
		}
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(env.Environment().GraphEndpoint)
	if err != nil {
		return nil, err
	}

	c := &Cluster{
		log: log,
		env: env,
		ci:  ci,

		deployments:                       features.NewDeploymentsClient(env.Environment(), env.SubscriptionID(), authorizer),
		groups:                            features.NewResourceGroupsClient(env.Environment(), env.SubscriptionID(), authorizer),
		openshiftclustersv20200430:        redhatopenshift20200430.NewOpenShiftClustersClient(env.Environment(), env.SubscriptionID(), authorizer),
		openshiftclustersv20210131preview: redhatopenshift20210131preview.NewOpenShiftClustersClient(env.Environment(), env.SubscriptionID(), authorizer),
		applications:                      graphrbac.NewApplicationsClient(env.Environment(), env.TenantID(), graphAuthorizer),
		serviceprincipals:                 graphrbac.NewServicePrincipalClient(env.Environment(), env.TenantID(), graphAuthorizer),
		securitygroups:                    network.NewSecurityGroupsClient(env.Environment(), env.SubscriptionID(), authorizer),
		subnets:                           network.NewSubnetsClient(env.Environment(), env.SubscriptionID(), authorizer),
		routetables:                       network.NewRouteTablesClient(env.Environment(), env.SubscriptionID(), authorizer),
		roleassignments:                   authorization.NewRoleAssignmentsClient(env.Environment(), env.SubscriptionID(), authorizer),
		peerings:                          network.NewVirtualNetworkPeeringsClient(env.Environment(), env.SubscriptionID(), authorizer),
	}

	if ci {
		if env.IsLocalDevelopmentMode() {
			c.ciParentVnet = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet", c.env.SubscriptionID(), c.env.ResourceGroup())
		} else {
			// This is dirty, but it used to be hard coded only for pub cloud.
			// TODO pick right config value to get sub and resource group
			if env.Environment().Name == azureclient.USGovernmentCloud.Name {
				c.ciParentVnet = "/subscriptions/28015960-ee66-4844-8037-fc28b0560bf1/resourceGroups/e2einfra-usgovvirginia/providers/Microsoft.Network/virtualNetworks/dev-vnet"
			} else {
				// default to prior behavior, public cloud int
				c.ciParentVnet = "/subscriptions/46626fc5-476d-41ad-8c76-2ec49c6994eb/resourceGroups/e2einfra-eastus/providers/Microsoft.Network/virtualNetworks/dev-vnet"
			}
		}

		r, err := azure.ParseResourceID(c.ciParentVnet)
		if err != nil {
			return nil, err
		}

		c.ciParentVnetPeerings = network.NewVirtualNetworkPeeringsClient(env.Environment(), r.SubscriptionID, authorizer)
	}

	return c, nil
}

func (c *Cluster) Create(ctx context.Context, vnetResourceGroup, clusterName string) error {
	_, err := c.openshiftclustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
	if err == nil {
		c.log.Print("cluster already exists, skipping create")
		return nil
	}

	fpSPID := os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")

	if fpSPID == "" {
		return fmt.Errorf("service principal id is not found")
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

	if c.ci {
		c.log.Infof("creating resource group")
		_, err = c.groups.CreateOrUpdate(ctx, vnetResourceGroup, mgmtfeatures.ResourceGroup{
			Location: to.StringPtr(c.env.Location()),
		})
		if err != nil {
			return err
		}

		visibility = api.VisibilityPrivate
	}

	b, err := deploy.Asset(generator.FileClusterPredeploy)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	addressPrefix, masterSubnet, workerSubnet := c.generateSubnets()
	if err != nil {
		return err
	}

	parameters := map[string]*arm.ParametersParameter{
		"clusterName":               {Value: clusterName},
		"ci":                        {Value: c.ci},
		"clusterServicePrincipalId": {Value: spID},
		"fpServicePrincipalId":      {Value: fpSPID},
		"vnetAddressPrefix":         {Value: addressPrefix},
		"masterAddressPrefix":       {Value: masterSubnet},
		"workerAddressPrefix":       {Value: workerSubnet},
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

	c.log.Info("creating role assignments")
	for _, scope := range []string{
		"/subscriptions/" + c.env.SubscriptionID() + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/dev-vnet",
		"/subscriptions/" + c.env.SubscriptionID() + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/routeTables/" + clusterName + "-rt",
	} {
		for _, principalID := range []string{spID, fpSPID} {
			for i := 0; i < 5; i++ {
				_, err = c.roleassignments.Create(
					ctx,
					scope,
					uuid.Must(uuid.NewV4()).String(),
					mgmtauthorization.RoleAssignmentCreateParameters{
						RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
							RoleDefinitionID: to.StringPtr("/subscriptions/" + c.env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleNetworkContributor),
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
	err = c.createCluster(ctx, vnetResourceGroup, clusterName, appID, appSecret, visibility)
	if err != nil {
		return err
	}

	if c.ci {
		c.log.Info("fixing up NSGs")
		err = c.fixupNSGs(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			return err
		}

		c.log.Info("peering subnets to CI infra")
		err = c.peerSubnetsToCI(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			return err
		}
	}

	c.log.Info("done")
	return nil
}

func (c *Cluster) generateSubnets() (vnetPrefix string, masterSubnet string, workerSubnet string) {
	// pick a random /23 in the range [10.0.2.0, 10.128.0.0).  10.0.0.0 is used
	// by dev-vnet to host CI; 10.128.0.0+ is used for pods.
	var x, y int
	for x == 0 && y == 0 {
		x, y = rand.Intn(128), 2*rand.Intn(128)
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

		r, err := azure.ParseResourceID(c.ciParentVnet)
		if err == nil {
			err = c.ciParentVnetPeerings.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, vnetResourceGroup+"-peer")
		}
		if err != nil {
			errs = append(errs, err)
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
func (c *Cluster) createCluster(ctx context.Context, vnetResourceGroup, clusterName, clientID, clientSecret string, visibility api.Visibility) error {
	// using internal representation for "singe source" of options
	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain:          strings.ToLower(clusterName),
				ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.env.SubscriptionID(), "aro-"+clusterName),
			},
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID:     clientID,
				ClientSecret: api.SecureString(clientSecret),
			},
			NetworkProfile: api.NetworkProfile{
				PodCIDR:     "10.128.0.0/14",
				ServiceCIDR: "172.30.0.0/16",
			},
			MasterProfile: api.MasterProfile{
				VMSize:   api.VMSizeStandardD8sV3,
				SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.env.SubscriptionID(), vnetResourceGroup, clusterName),
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:       "worker",
					VMSize:     api.VMSizeStandardD4sV3,
					DiskSizeGB: 128,
					SubnetID:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", c.env.SubscriptionID(), vnetResourceGroup, clusterName),
					Count:      3,
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

		oc.Properties.WorkerProfiles[0].VMSize = api.VMSizeStandardD2sV3
		ext := api.APIs[v20210131preview.APIVersion].OpenShiftClusterConverter().ToExternal(&oc)
		data, err := json.Marshal(ext)
		if err != nil {
			return err
		}

		ocExt := mgmtredhatopenshift20210131preview.OpenShiftCluster{}
		err = json.Unmarshal(data, &ocExt)
		if err != nil {
			return err
		}

		return c.openshiftclustersv20210131preview.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, ocExt)

	} else {
		ext := api.APIs[v20200430.APIVersion].OpenShiftClusterConverter().ToExternal(&oc)
		data, err := json.Marshal(ext)
		if err != nil {
			return err
		}

		ocExt := mgmtredhatopenshift20200430.OpenShiftCluster{}
		err = json.Unmarshal(data, &ocExt)
		if err != nil {
			return err
		}

		return c.openshiftclustersv20200430.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, ocExt)
	}
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
	}
	rpProp := &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &mgmtnetwork.SubResource{
			ID: &cluster,
		},
		AllowVirtualNetworkAccess: to.BoolPtr(true),
		AllowForwardedTraffic:     to.BoolPtr(true),
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

func (c *Cluster) deleteRoleAssignments(ctx context.Context, vnetResourceGroup, appID string) error {
	spObjID, err := c.getServicePrincipal(ctx, appID)
	if err != nil {
		return err
	}
	if spObjID == "" {
		return nil
	}

	roleAssignments, err := c.roleassignments.ListForResourceGroup(ctx, vnetResourceGroup, fmt.Sprintf("principalId eq '%s'", spObjID))
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
