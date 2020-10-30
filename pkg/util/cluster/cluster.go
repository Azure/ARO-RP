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

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	mgmtredhatopenshift "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
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
	subnets           network.SubnetsClient
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
		subnets:           network.NewSubnetsClient(instancemetadata.SubscriptionID(), authorizer),
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

	// HACK: NRM
	if c.ci {
		c.log.Info("waiting for and detaching NSGs")
		err = c.waitForAndDetachNSGs(ctx, clusterName)
		if err != nil {
			return err
		}
	}

	c.log.Info("done")
	return nil
}

func (c *Cluster) Delete(ctx context.Context, clusterName string) error {
	oc, err := c.openshiftclusters.Get(ctx, c.ResourceGroup(), clusterName)
	if err == nil {
		err = c.deleteApplication(ctx, *oc.OpenShiftClusterProperties.ServicePrincipalProfile.ClientID)
		if err != nil {
			c.log.Warn(err)
		}

		c.log.Print("deleting cluster")
		err = c.openshiftclusters.DeleteAndWait(ctx, c.ResourceGroup(), clusterName)
		if err != nil {
			c.log.Warn(err)
		}
	}

	if c.deploymentMode == deployment.Development {
		c.log.Info("detaching NSGs")
		for _, subnetName := range []string{
			clusterName + "-master",
			clusterName + "-worker",
		} {
			err = c.detachNSG(ctx, clusterName, subnetName)
			if err != nil {
				c.log.Warn(err)
			}
		}

		_, err = c.groups.Get(ctx, "aro-"+clusterName)
		if err == nil {
			c.log.Print("deleting cluster resource group (belt and braces)")
			err = c.groups.DeleteAndWait(ctx, "aro-"+clusterName)
			if err != nil {
				c.log.Warn(err)
			}
		}
	}

	if c.ci {
		_, err = c.groups.Get(ctx, c.ResourceGroup())
		if err == nil {
			c.log.Print("deleting resource group")
			err = c.groups.DeleteAndWait(ctx, c.ResourceGroup())
			if err != nil {
				c.log.Warn(err)
			}
		}
	} else {
		// TODO: clean up subnets, route table, RBAC
	}

	c.log.Info("done")
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

func (c *Cluster) waitForAndDetachNSGs(ctx context.Context, clusterName string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, subnetName := range []string{
			clusterName + "-master",
			clusterName + "-worker",
		} {
			subnet, err := c.subnets.Get(ctx, c.ResourceGroup(), "dev-vnet", subnetName, "")
			if err != nil {
				return false, err
			}
			err = c.detachNSG(ctx, clusterName, subnetName)
			if err != nil {
				return false, err
			}

			if subnet.NetworkSecurityGroup != nil {
				return false, nil
			}
		}

		return true, nil
	}, ctx.Done())
	return err
}

func (c *Cluster) detachNSG(ctx context.Context, clusterName, subnetName string) error {
	subnet, err := c.subnets.Get(ctx, c.ResourceGroup(), "dev-vnet", subnetName, "")
	if err != nil {
		return err
	}

	if subnet.NetworkSecurityGroup == nil {
		return nil
	}

	subnet.NetworkSecurityGroup = nil

	return c.subnets.CreateOrUpdateAndWait(ctx, c.ResourceGroup(), "dev-vnet", subnetName, subnet)
}
