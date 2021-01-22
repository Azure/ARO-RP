package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	v2021131preview "github.com/Azure/ARO-RP/pkg/api/v20210131preview"
	mgmtopenshiftclustersv20200430 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	mgmtopenshiftclustersv20210131preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	openshiftclustersv20200430 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2020-04-30/redhatopenshift"
	openshiftclustersv20210131preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Cluster struct {
	log        *logrus.Entry
	localEnv   env.Core
	clusterEnv env.Core
	ci         bool

	deployments                       features.DeploymentsClient
	groups                            features.ResourceGroupsClient
	applications                      graphrbac.ApplicationsClient
	serviceprincipals                 graphrbac.ServicePrincipalClient
	openshiftclustersv20200430        openshiftclustersv20200430.OpenShiftClustersClient
	openshiftclustersv20210131preview openshiftclustersv20210131preview.OpenShiftClustersClient
	securitygroups                    network.SecurityGroupsClient
	subnets                           network.SubnetsClient
	routetables                       network.RouteTablesClient
	roleassignments                   authorization.RoleAssignmentsClient
}

const (
	firstPartyClientIDProduction  = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"
	firstPartyClientIDIntegration = "71cfb175-ea3a-444e-8c03-b119b2752ce4"
)

type errors []error

func (errs errors) Error() string {
	var sb strings.Builder

	for _, err := range errs {
		sb.WriteString(err.Error())
		sb.WriteByte('\n')
	}

	return sb.String()
}

func New(log *logrus.Entry, localEnv env.Core, clusterEnv env.Core, ci bool) (*Cluster, error) {
	if clusterEnv.DeploymentMode() == deployment.Development {
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

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(localEnv.Environment().GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		log:        log,
		clusterEnv: clusterEnv,
		localEnv:   localEnv,
		ci:         ci,

		deployments:                       features.NewDeploymentsClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		groups:                            features.NewResourceGroupsClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		openshiftclustersv20200430:        openshiftclustersv20200430.NewOpenShiftClustersClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		openshiftclustersv20210131preview: openshiftclustersv20210131preview.NewOpenShiftClustersClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		applications:                      graphrbac.NewApplicationsClient(localEnv.Environment(), localEnv.TenantID(), graphAuthorizer),
		serviceprincipals:                 graphrbac.NewServicePrincipalClient(localEnv.Environment(), localEnv.TenantID(), graphAuthorizer),
		securitygroups:                    network.NewSecurityGroupsClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		subnets:                           network.NewSubnetsClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		routetables:                       network.NewRouteTablesClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
		roleassignments:                   authorization.NewRoleAssignmentsClient(localEnv.Environment(), localEnv.SubscriptionID(), authorizer),
	}, nil
}

func (c *Cluster) Create(ctx context.Context, clusterName string) error {
	_, err := c.openshiftclustersv20200430.Get(ctx, c.clusterEnv.ResourceGroup(), clusterName)
	if err == nil {
		c.log.Print("cluster already exists, skipping create")
		return nil
	}

	var fpClientID string
	switch c.clusterEnv.DeploymentMode() {
	case deployment.Integration:
		fpClientID = firstPartyClientIDIntegration
	case deployment.Production:
		fpClientID = firstPartyClientIDProduction
	default:
		fpClientID = os.Getenv("AZURE_FP_CLIENT_ID")
	}

	fpSPID, err := c.getServicePrincipal(ctx, fpClientID)
	if err != nil {
		return err
	}
	if fpSPID == "" {
		return fmt.Errorf("service principal not found for appId %s", fpClientID)
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

	if c.ci {
		c.log.Infof("creating resource group")
		_, err = c.groups.CreateOrUpdate(ctx, c.clusterEnv.ResourceGroup(), mgmtfeatures.ResourceGroup{
			Location: to.StringPtr(c.clusterEnv.Location()),
		})
		if err != nil {
			return err
		}
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

	parameters := map[string]*arm.ParametersParameter{
		"clusterName":               {Value: clusterName},
		"clusterServicePrincipalId": {Value: spID},
		"fpServicePrincipalId":      {Value: fpSPID},
		"fullDeploy":                {Value: c.ci},
		"masterAddressPrefix":       {Value: fmt.Sprintf("10.%d.%d.0/24", rand.Intn(128), rand.Intn(256))},
		"workerAddressPrefix":       {Value: fmt.Sprintf("10.%d.%d.0/24", rand.Intn(128), rand.Intn(256))},
	}

	armctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	c.log.Info("predeploying ARM template")
	err = c.deployments.CreateOrUpdateAndWait(armctx, c.clusterEnv.ResourceGroup(), clusterName, mgmtfeatures.Deployment{
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
		"/subscriptions/" + c.clusterEnv.SubscriptionID() + "/resourceGroups/" + c.clusterEnv.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/dev-vnet",
		"/subscriptions/" + c.clusterEnv.SubscriptionID() + "/resourceGroups/" + c.clusterEnv.ResourceGroup() + "/providers/Microsoft.Network/routeTables/" + clusterName + "-rt",
	} {
		for _, principalID := range []string{spID, fpSPID} {
			for i := 0; i < 5; i++ {
				_, err = c.roleassignments.Create(
					ctx,
					scope,
					uuid.NewV4().String(),
					mgmtauthorization.RoleAssignmentCreateParameters{
						RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
							RoleDefinitionID: to.StringPtr("/subscriptions/" + c.clusterEnv.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleNetworkContributor),
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
	err = c.createCluster(ctx, clusterName, appID, appSecret)
	if err != nil {
		return err
	}

	if c.ci {
		c.log.Info("fixing up NSGs")
		err = c.fixupNSGs(ctx, clusterName)
		if err != nil {
			return err
		}
	}

	c.log.Info("done")
	return nil
}

func (c *Cluster) Delete(ctx context.Context, clusterName string) error {
	var errs errors

	oc, err := c.openshiftclustersv20200430.Get(ctx, c.clusterEnv.ResourceGroup(), clusterName)
	if err == nil {
		err = c.deleteRoleAssignments(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}

		err = c.deleteApplication(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Print("deleting cluster")
		err = c.openshiftclustersv20200430.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup(), clusterName)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if c.ci {
		_, err = c.groups.Get(ctx, c.clusterEnv.ResourceGroup())
		if err == nil {
			c.log.Print("deleting resource group")
			err = c.groups.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup())
			if err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		// Deleting the deployment does not clean up the associated resources
		c.log.Info("deleting deployment")
		err = c.deployments.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup(), clusterName)
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting master/worker subnets")
		err = c.subnets.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup(), "dev-vnet", clusterName+"-master")
		if err != nil {
			errs = append(errs, err)
		}

		err = c.subnets.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup(), "dev-vnet", clusterName+"-worker")
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting route table")
		err = c.routetables.DeleteAndWait(ctx, c.clusterEnv.ResourceGroup(), clusterName+"-rt")
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
func (c *Cluster) createCluster(ctx context.Context, clusterName, clientID, clientSecret string) error {
	// using internal representation for "singe source" of options
	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain:          strings.ToLower(clusterName),
				ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.clusterEnv.SubscriptionID(), "aro-"+clusterName),
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
				SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.clusterEnv.SubscriptionID(), c.clusterEnv.ResourceGroup(), clusterName),
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:       "worker",
					VMSize:     api.VMSizeStandardD4sV3,
					DiskSizeGB: 128,
					SubnetID:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", c.clusterEnv.SubscriptionID(), c.clusterEnv.ResourceGroup(), clusterName),
					Count:      3,
				},
			},
			APIServerProfile: api.APIServerProfile{
				Visibility: api.VisibilityPublic,
			},
			IngressProfiles: []api.IngressProfile{
				{
					Name:       "default",
					Visibility: api.VisibilityPublic,
				},
			},
		},
		Location: c.clusterEnv.Location(),
	}

	switch c.clusterEnv.DeploymentMode() {
	case deployment.Development:
		oc.Properties.WorkerProfiles[0].VMSize = api.VMSizeStandardD2sV3
		ext := api.APIs[v2021131preview.APIVersion].OpenShiftClusterConverter().ToExternal(&oc)
		data, err := json.Marshal(ext)
		if err != nil {
			return err
		}

		ocExt := mgmtopenshiftclustersv20210131preview.OpenShiftCluster{}
		err = json.Unmarshal(data, &ocExt)
		if err != nil {
			return err
		}

		return c.openshiftclustersv20210131preview.CreateOrUpdateAndWait(ctx, c.clusterEnv.ResourceGroup(), clusterName, ocExt)
	default:
		ext := api.APIs[v20200430.APIVersion].OpenShiftClusterConverter().ToExternal(&oc)
		data, err := json.Marshal(ext)
		if err != nil {
			return err
		}

		ocExt := mgmtopenshiftclustersv20200430.OpenShiftCluster{}
		err = json.Unmarshal(data, &ocExt)
		if err != nil {
			return err
		}

		return c.openshiftclustersv20200430.CreateOrUpdateAndWait(ctx, c.clusterEnv.ResourceGroup(), clusterName, ocExt)
	}
}

func (c *Cluster) fixupNSGs(ctx context.Context, clusterName string) error {
	// TODO: simplify after 4.5 is rolled out.

	type fix struct {
		subnetName string
		nsgID      string
	}

	var fixes []*fix

	nsgs, err := c.securitygroups.List(ctx, "aro-"+clusterName)
	if err != nil {
		return err
	}

	if len(nsgs) == 2 {
		// ArchitectureVersionV1
		for _, nsg := range nsgs {
			switch {
			case strings.HasSuffix(*nsg.Name, subnet.NSGControlPlaneSuffixV1):
				fixes = append(fixes, &fix{
					subnetName: clusterName + "-master",
					nsgID:      *nsg.ID,
				})

			case strings.HasSuffix(*nsg.Name, subnet.NSGNodeSuffixV1):
				fixes = append(fixes, &fix{
					subnetName: clusterName + "-worker",
					nsgID:      *nsg.ID,
				})
			}
		}

	} else {
		// ArchitectureVersionV2
		fixes = []*fix{
			{
				subnetName: clusterName + "-master",
				nsgID:      *nsgs[0].ID,
			},
			{
				subnetName: clusterName + "-worker",
				nsgID:      *nsgs[0].ID,
			},
		}
	}

	for _, fix := range fixes {
		subnet, err := c.subnets.Get(ctx, c.clusterEnv.ResourceGroup(), "dev-vnet", fix.subnetName, "")
		if err != nil {
			return err
		}

		subnet.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: &fix.nsgID,
		}

		err = c.subnets.CreateOrUpdateAndWait(ctx, c.clusterEnv.ResourceGroup(), "dev-vnet", fix.subnetName, subnet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) deleteRoleAssignments(ctx context.Context, appID string) error {
	spObjID, err := c.getServicePrincipal(ctx, appID)
	if err != nil {
		return err
	}
	if spObjID == "" {
		return nil
	}

	roleAssignments, err := c.roleassignments.ListForResourceGroup(ctx, c.clusterEnv.ResourceGroup(), fmt.Sprintf("principalId eq '%s'", spObjID))
	if err != nil {
		return err
	}

	for _, roleAssignment := range roleAssignments {
		c.log.Infof("deleting role assignment %s", *roleAssignment.Name)
		_, err = c.roleassignments.Delete(ctx, *roleAssignment.Scope, *roleAssignment.Name)
		if err != nil {
			return err
		}
	}

	return nil
}
