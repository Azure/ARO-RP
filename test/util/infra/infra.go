package infra

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	mgmtgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	mgmtaro "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

var fpClientID = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"

var _ Interface = &Infrastructure{}

// Interface abstracts infrastructure creation for ARO
type Interface interface {
	Deploy(context.Context) error
	Destroy(context.Context) error
}

type Infrastructure struct {
	log *logrus.Entry

	clusterName          string
	clusterResourceGroup string
	resourceGroup        string
	location             string
	subscriptionID       string
	postfix              string

	deployments       features.DeploymentsClient
	groups            features.ResourceGroupsClient
	applications      graphrbac.ApplicationsClient
	serviceprincipals graphrbac.ServicePrincipalClient
	openshiftclusters mgmtaro.OpenShiftClustersClient

	appOID *string // used to delete the app
}

// New created new test infrastructure object
func New(log *logrus.Entry, subscriptionID, tenantID string) (*Infrastructure, error) {
	authorizer, graphAuthorizer, err := getAuthorizers(log)
	if err != nil {
		return nil, err
	}

	return &Infrastructure{
		log:                  log,
		clusterName:          os.Getenv("CLUSTER"),
		clusterResourceGroup: "aro-" + os.Getenv("RESOURCEGROUP"),
		resourceGroup:        os.Getenv("RESOURCEGROUP"),
		location:             os.Getenv("LOCATION"),
		subscriptionID:       subscriptionID,
		postfix:              strconv.FormatInt(time.Now().Unix(), 10), // set once so all componets would use same value

		deployments:       features.NewDeploymentsClient(subscriptionID, authorizer),
		groups:            features.NewResourceGroupsClient(subscriptionID, authorizer),
		openshiftclusters: mgmtaro.NewOpenShiftClustersClient(subscriptionID, authorizer),
		applications:      graphrbac.NewApplicationsClient(tenantID, graphAuthorizer),
		serviceprincipals: graphrbac.NewServicePrincipalClient(tenantID, graphAuthorizer),
	}, nil

}

// getAuthorizers returns authorizers based on where are running. This helps us
// to avoid setting env variables in CI and prevent any potential leaks with bad
// scripting.
// If E2E variable is not set we are running in CI with CLI context.
// If it is set, we ignore CLI context and use ENV
func getAuthorizers(log *logrus.Entry) (autorest.Authorizer, autorest.Authorizer, error) {
	if os.Getenv("E2E_CREATE_CLUSTER") != "" {
		log.Info("authorizer from CLI")
		authorizer, err := auth.NewAuthorizerFromCLI()
		if err != nil {
			return nil, nil, err
		}
		graphAuthorizer, err := auth.NewAuthorizerFromCLIWithResource(azure.PublicCloud.GraphEndpoint)
		return authorizer, graphAuthorizer, err

	}
	log.Info("authorizer from ENV")
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, nil, err
	}

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.GraphEndpoint)
	return authorizer, graphAuthorizer, err
}

func (i *Infrastructure) Deploy(ctx context.Context) error {
	if os.Getenv("RP_MODE") != "" {
		fpClientID = os.Getenv("AZURE_FP_CLIENT_ID")
	}

	_, err := i.openshiftclusters.Get(ctx, i.resourceGroup, i.resourceGroup)
	if err == nil {
		i.log.Warn("Cluster already exist, skiping create")
		return nil
	}

	// TODO: we are listing here rather than calling
	// i.applications.GetServicePrincipalsIDByAppID() due to some missing
	// permission with our dev/e2e applications
	results, err := i.serviceprincipals.List(ctx, fmt.Sprintf("appId eq '%s'", fpClientID))
	if err != nil {
		return err
	}
	if len(results) != 1 {
		return fmt.
			Errorf("More than one application found for FP AppId")
	}
	fpSPID := *results[0].ObjectID

	i.log.Infof("Create resource group %s", i.resourceGroup)
	_, err = i.groups.CreateOrUpdate(ctx, i.resourceGroup, mgmtfeatures.ResourceGroup{Location: &i.location})
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-%s-%s", i.clusterName, i.location, i.postfix)
	i.log.Infof("Create aad app %s", name)
	appID, appSecret, spID, err := i.deployAAD(ctx, name)
	if err != nil {
		return err
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
		"clusterName":               {Value: i.clusterName},
		"clusterServicePrincipalId": {Value: spID},
		"fpServicePrincipalId":      {Value: fpSPID},
		"fullDeploy":                {Value: true},
		"masterAddressPrefix":       {Value: fmt.Sprintf("10.%d.%d.0/24", rand.Intn(128), rand.Intn(256))},
		"workerAddressPrefix":       {Value: fmt.Sprintf("10.%d.%d.0/24", rand.Intn(128), rand.Intn(256))},
	}

	i.log.Info("Create ci-infra arm")
	err = i.deployARMTemplate(ctx, template, parameters)
	if err != nil {
		return err
	}

	i.log.Infof("Create a cluster %s", i.clusterName)
	return i.deployCluster(ctx, appID, appSecret)
}

func (i *Infrastructure) Destroy(ctx context.Context) error {
	i.log.Infof("destroy infrastructure %s/%s", i.resourceGroup, i.clusterName)
	err := i.openshiftclusters.DeleteAndWait(ctx, i.resourceGroup, i.clusterName)
	if err != nil {
		return err
	}

	if i.appOID != nil {
		_, err = i.applications.Delete(ctx, *i.appOID)
		if err != nil {
			return err
		}
	}

	return i.groups.DeleteAndWait(ctx, i.resourceGroup)
}

// Deploy puts the cloud infra in place using ARM template and Deploy method
func (i *Infrastructure) deployARMTemplate(ctx context.Context, t interface{}, parameters map[string]*arm.ParametersParameter) error {
	err := i.deployments.CreateOrUpdateAndWait(ctx, i.resourceGroup, "ci-infra", mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   t,
			Parameters: parameters,
			Mode:       mgmtfeatures.Incremental,
		},
	})

	if azureerrors.IsDeploymentActiveError(err) {
		err = i.deployments.Wait(ctx, i.resourceGroup, "ci-infra")
	}

	return err
}

func (i *Infrastructure) deployAAD(ctx context.Context, name string) (appID string, appSecret string, spID string, err error) {
	t := time.Now().UTC().Truncate(time.Second)

	appStartdate := date.Time{Time: t}
	appEndDate := date.Time{Time: t.Add(48 * time.Hour)}
	passID := uuid.NewV4()
	pass := uuid.NewV4()

	app, err := i.applications.Create(ctx, mgmtgraphrbac.ApplicationCreateParameters{
		DisplayName:             to.StringPtr(name),
		Homepage:                to.StringPtr("https://" + name),
		IdentifierUris:          &[]string{"https://" + name},
		AvailableToOtherTenants: to.BoolPtr(false),
		PasswordCredentials: &[]mgmtgraphrbac.PasswordCredential{
			{
				StartDate: &appStartdate,
				EndDate:   &appEndDate,
				KeyID:     to.StringPtr(passID.String()),
				Value:     to.StringPtr(pass.String()),
			},
		},
		ReplyUrls: &[]string{},
	})
	i.appOID = app.ObjectID // need to be set for cleaning

	if err != nil {
		return "", "", "", err
	}

	sp, err := i.serviceprincipals.Create(ctx, mgmtgraphrbac.ServicePrincipalCreateParameters{
		AppID:          app.AppID,
		AccountEnabled: to.BoolPtr(true),
	})

	if err != nil {
		return "", "", "", err
	}

	return *app.AppID, pass.String(), *sp.ObjectID, nil
}

func (i *Infrastructure) deployCluster(ctx context.Context, spID, secret string) error {
	cluster := redhatopenshift.OpenShiftCluster{
		Location: &i.location,
		OpenShiftClusterProperties: &redhatopenshift.OpenShiftClusterProperties{
			ClusterProfile: &redhatopenshift.ClusterProfile{
				Domain: to.StringPtr("v4-" + i.postfix),
			},
			ServicePrincipalProfile: &redhatopenshift.ServicePrincipalProfile{
				ClientID:     to.StringPtr(spID),
				ClientSecret: to.StringPtr(secret),
			},
			NetworkProfile: &redhatopenshift.NetworkProfile{
				PodCidr:     to.StringPtr("10.128.0.0/14"),
				ServiceCidr: to.StringPtr("172.30.0.0/16"),
			},
			MasterProfile: &redhatopenshift.MasterProfile{
				VMSize:   redhatopenshift.StandardD8sV3,
				SubnetID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", i.subscriptionID, i.resourceGroup, i.clusterName)),
			},
			WorkerProfiles: &[]redhatopenshift.WorkerProfile{
				{
					Name:       to.StringPtr("worker"),
					Count:      to.Int32Ptr(3),
					DiskSizeGB: to.Int32Ptr(128),
					VMSize:     redhatopenshift.VMSize1StandardD2sV3,
					SubnetID:   to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", i.subscriptionID, i.resourceGroup, i.clusterName)),
				},
			},
			ApiserverProfile: &redhatopenshift.APIServerProfile{
				Visibility: redhatopenshift.Public,
			},
			IngressProfiles: &[]redhatopenshift.IngressProfile{
				{
					Visibility: redhatopenshift.Visibility1Public,
					Name:       to.StringPtr("default"),
				},
			},
		},
	}

	if i.clusterResourceGroup != "" {
		cluster.ClusterProfile.ResourceGroupID = to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", i.subscriptionID, i.clusterResourceGroup))
	}
	// D2sV3 is not supported in production
	if os.Getenv("RP_MODE") == "" {
		(*cluster.WorkerProfiles)[0].VMSize = redhatopenshift.VMSize1StandardD4sV3
	}

	return i.openshiftclusters.CreateOrUpdateAndWait(ctx, i.resourceGroup, i.clusterName, cluster)
}
