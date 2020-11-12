package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	mgmtredhatopenshift "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Cluster struct {
	log            *logrus.Entry
	deploymentMode deployment.Mode
	instancemetadata.InstanceMetadata
	ci bool

	deployments       features.DeploymentsClient
	groups            features.ResourceGroupsClient
	applications      graphrbac.ApplicationsClient
	serviceprincipals graphrbac.ServicePrincipalClient
	openshiftclusters redhatopenshift.OpenShiftClustersClient
	securitygroups    network.SecurityGroupsClient
	subnets           network.SubnetsClient
	routetables       network.RouteTablesClient
	roleassignments   authorization.RoleAssignmentsClient
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

func New(log *logrus.Entry, deploymentMode deployment.Mode, instancemetadata instancemetadata.InstanceMetadata, ci bool) (*Cluster, error) {
	if deploymentMode != deployment.Production {
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

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		log:              log,
		deploymentMode:   deploymentMode,
		InstanceMetadata: instancemetadata,
		ci:               ci,

		deployments:       features.NewDeploymentsClient(instancemetadata.SubscriptionID(), authorizer),
		groups:            features.NewResourceGroupsClient(instancemetadata.SubscriptionID(), authorizer),
		openshiftclusters: redhatopenshift.NewOpenShiftClustersClient(instancemetadata.SubscriptionID(), authorizer),
		applications:      graphrbac.NewApplicationsClient(instancemetadata.TenantID(), graphAuthorizer),
		serviceprincipals: graphrbac.NewServicePrincipalClient(instancemetadata.TenantID(), graphAuthorizer),
		securitygroups:    network.NewSecurityGroupsClient(instancemetadata.SubscriptionID(), authorizer),
		subnets:           network.NewSubnetsClient(instancemetadata.SubscriptionID(), authorizer),
		routetables:       network.NewRouteTablesClient(instancemetadata.SubscriptionID(), authorizer),
		roleassignments:   authorization.NewRoleAssignmentsClient(instancemetadata.SubscriptionID(), authorizer),
	}, nil
}

func (c *Cluster) Create(ctx context.Context, clusterName string) error {
	_, err := c.openshiftclusters.Get(ctx, c.ResourceGroup(), clusterName)
	if err == nil {
		c.log.Print("cluster already exists, skipping create")
		return nil
	}

	fpClientID := "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"
	if c.deploymentMode != deployment.Production {
		fpClientID = os.Getenv("AZURE_FP_CLIENT_ID")
	}

	fpSPID, err := c.getServicePrincipal(ctx, fpClientID)
	if err != nil {
		return err
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
		_, err = c.groups.CreateOrUpdate(ctx, c.ResourceGroup(), mgmtfeatures.ResourceGroup{
			Location: to.StringPtr(c.Location()),
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
	err = c.deployments.CreateOrUpdateAndWait(armctx, c.ResourceGroup(), clusterName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Parameters: parameters,
			Mode:       mgmtfeatures.Incremental,
		},
	})
	if err != nil {
		return err
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

	oc, err := c.openshiftclusters.Get(ctx, c.ResourceGroup(), clusterName)
	if err == nil {
		c.log.Print("deleting cluster")
		err = c.openshiftclusters.DeleteAndWait(ctx, c.ResourceGroup(), clusterName)
		if err != nil {
			errs = append(errs, err)
		}

		err = c.deleteApplication(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}

		// Cluster service principal has contributor access on clusterResourceGroup.
		// At the same time ClusterResourceGroup is protected with DenyAssignment,
		// so this can't be deleted while cluster still exists.
		err = c.deleteRoleAssignments(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if c.ci {
		_, err = c.groups.Get(ctx, c.ResourceGroup())
		if err == nil {
			c.log.Print("deleting resource group")
			err = c.groups.DeleteAndWait(ctx, c.ResourceGroup())
			if err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		c.log.Info("removing predeployment resources")

		// Deleting the deployment does not clean up the associated resources
		c.log.Info("deleting deployment")
		err = c.deployments.DeleteAndWait(ctx, c.ResourceGroup(), clusterName)
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting master/worker subnets")
		err = c.subnets.DeleteAndWait(ctx, c.ResourceGroup(), "dev-vnet", clusterName+"-worker")
		if err != nil {
			errs = append(errs, err)
		}

		err = c.subnets.DeleteAndWait(ctx, c.ResourceGroup(), "dev-vnet", clusterName+"-master")
		if err != nil {
			errs = append(errs, err)
		}

		c.log.Info("deleting route table")
		err = c.routetables.DeleteAndWait(ctx, c.ResourceGroup(), clusterName+"-rt")
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

func (c *Cluster) createCluster(ctx context.Context, clusterName, clientID, clientSecret string) error {
	oc := mgmtredhatopenshift.OpenShiftCluster{
		OpenShiftClusterProperties: &mgmtredhatopenshift.OpenShiftClusterProperties{
			ClusterProfile: &mgmtredhatopenshift.ClusterProfile{
				Domain:          to.StringPtr(strings.ToLower(clusterName)),
				ResourceGroupID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.SubscriptionID(), "aro-"+clusterName)),
			},
			ServicePrincipalProfile: &mgmtredhatopenshift.ServicePrincipalProfile{
				ClientID:     to.StringPtr(clientID),
				ClientSecret: to.StringPtr(clientSecret),
			},
			NetworkProfile: &mgmtredhatopenshift.NetworkProfile{
				PodCidr:     to.StringPtr("10.128.0.0/14"),
				ServiceCidr: to.StringPtr("172.30.0.0/16"),
			},
			MasterProfile: &mgmtredhatopenshift.MasterProfile{
				VMSize:   mgmtredhatopenshift.StandardD8sV3,
				SubnetID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.SubscriptionID(), c.ResourceGroup(), clusterName)),
			},
			WorkerProfiles: &[]mgmtredhatopenshift.WorkerProfile{
				{
					Name:       to.StringPtr("worker"),
					VMSize:     mgmtredhatopenshift.VMSize1StandardD4sV3,
					DiskSizeGB: to.Int32Ptr(128),
					SubnetID:   to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", c.SubscriptionID(), c.ResourceGroup(), clusterName)),
					Count:      to.Int32Ptr(3),
				},
			},
			ApiserverProfile: &mgmtredhatopenshift.APIServerProfile{
				Visibility: mgmtredhatopenshift.Public,
			},
			IngressProfiles: &[]mgmtredhatopenshift.IngressProfile{
				{
					Name:       to.StringPtr("default"),
					Visibility: mgmtredhatopenshift.Visibility1Public,
				},
			},
		},
		Location: to.StringPtr(c.Location()),
	}

	if c.deploymentMode == deployment.Development {
		(*oc.WorkerProfiles)[0].VMSize = mgmtredhatopenshift.VMSize1StandardD2sV3
	}

	return c.openshiftclusters.CreateOrUpdateAndWait(ctx, c.ResourceGroup(), clusterName, oc)
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
		subnet, err := c.subnets.Get(ctx, c.ResourceGroup(), "dev-vnet", fix.subnetName, "")
		if err != nil {
			return err
		}

		subnet.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: &fix.nsgID,
		}

		err = c.subnets.CreateOrUpdateAndWait(ctx, c.ResourceGroup(), "dev-vnet", fix.subnetName, subnet)
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

	roleAssignments, err := c.roleassignments.List(ctx, fmt.Sprintf("principalId eq '%s'", spObjID))
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
