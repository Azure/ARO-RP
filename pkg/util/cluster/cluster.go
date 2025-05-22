package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	armsdk "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	sdkkeyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy/assets"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armkeyvault"
	armmsiclient "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	redhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type ClusterConfig struct {
	ClusterName                    string `mapstructure:"CLUSTER"`
	SubscriptionID                 string `mapstructure:"AZURE_SUBSCRIPTION_ID"`
	TenantID                       string `mapstructure:"AZURE_TENANT_ID"`
	Location                       string `mapstructure:"LOCATION"`
	AzureEnvironment               string `mapstructure:"AZURE_ENVIRONMENT"`
	UseWorkloadIdentity            bool   `mapstructure:"USE_WI"`
	WorkloadIdentityRoles          string `mapstructure:"PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"`
	IdentityPoolResourceGroup      string `mapstructure:"PLATFORM_WORKLOAD_IDENTITY_POOL_RESOURCEGROUP"`
	IdentityPoolClaimDurationHours int    `mapstructure:"PLATFORM_WORKLOAD_IDENTITY_POOL_CLAIM_DURATION_HOURS"`
	IsCI                           bool   `mapstructure:"CI"`
	RpMode                         string `mapstructure:"RP_MODE"`
	VnetResourceGroup              string `mapstructure:"CLUSTER_RESOURCEGROUP"`
	RPResourceGroup                string `mapstructure:"RESOURCEGROUP"`
	OSClusterVersion               string `mapstructure:"OS_CLUSTER_VERSION"`
	FPServicePrincipalID           string `mapstructure:"AZURE_FP_SERVICE_PRINCIPAL_ID"`
	IsPrivate                      bool   `mapstructure:"PRIVATE_CLUSTER"`
	NoInternet                     bool   `mapstructure:"NO_INTERNET"`
	MockMSIObjectID                string `mapstructure:"MOCK_MSI_OBJECT_ID"`

	MasterVMSize string `mapstructure:"MASTER_VM_SIZE"`
	WorkerVMSize string `mapstructure:"WORKER_VM_SIZE"`
}

func (cc *ClusterConfig) IsLocalDevelopmentMode() bool {
	return strings.EqualFold(cc.RpMode, "development")
}

type Cluster struct {
	log                *logrus.Entry
	Config             *ClusterConfig
	ciParentVnet       string
	workloadIdentities map[string]api.PlatformWorkloadIdentity

	spGraphClient            *utilgraph.GraphServiceClient
	deployments              features.DeploymentsClient
	groups                   features.ResourceGroupsClient
	openshiftclusters        InternalClient
	securitygroups           armnetwork.SecurityGroupsClient
	subnets                  armnetwork.SubnetsClient
	routetables              armnetwork.RouteTablesClient
	roleassignments          authorization.RoleAssignmentsClient
	roledefinitions          authorization.RoleDefinitionsClient
	peerings                 armnetwork.VirtualNetworkPeeringsClient
	ciParentVnetPeerings     armnetwork.VirtualNetworkPeeringsClient
	vaultsClient             armkeyvault.VaultsClient
	msiClient                armmsiclient.UserAssignedIdentitiesClient
	diskEncryptionSetsClient compute.DiskEncryptionSetsClient
}

const GenerateSubnetMaxTries = 100
const localDefaultURL string = "https://localhost:8443"
const DefaultMasterVmSize = api.VMSizeStandardD8sV5
const DefaultWorkerVmSize = api.VMSizeStandardD4sV5

func insecureLocalClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func NewClusterConfigFromEnv() (*ClusterConfig, error) {
	var conf ClusterConfig
	viper.AutomaticEnv()
	viper.SetOptions(viper.ExperimentalBindStruct())
	err := viper.Unmarshal(&conf)

	if err != nil {
		return nil, fmt.Errorf("error parsing env vars: %w", err)
	}

	if conf.ClusterName == "" {
		return nil, fmt.Errorf("cluster Name must be set")
	}

	if conf.UseWorkloadIdentity && conf.WorkloadIdentityRoles == "" {
		return nil, fmt.Errorf("workload Identity Role Set must be set")
	}

	if conf.RPResourceGroup == "" {
		return nil, fmt.Errorf("resource group must be set")
	}

	if conf.FPServicePrincipalID == "" {
		return nil, fmt.Errorf("fP Service Principal ID must be set")
	}

	if conf.IsCI {
		conf.VnetResourceGroup = conf.ClusterName
	} else {
		conf.VnetResourceGroup = conf.RPResourceGroup
	}

	if !conf.IsCI && conf.VnetResourceGroup == "" {
		return nil, fmt.Errorf("resource Group must be set")
	}

	if conf.OSClusterVersion == "" {
		conf.OSClusterVersion = version.DefaultInstallStream.Version.String()
	}

	if conf.AzureEnvironment == "" {
		conf.AzureEnvironment = "AZUREPUBLICCLOUD"
	}

	if conf.MasterVMSize == "" {
		conf.MasterVMSize = DefaultMasterVmSize.String()
	}
	if conf.WorkerVMSize == "" {
		conf.WorkerVMSize = DefaultWorkerVmSize.String()
	}

	if conf.IdentityPoolResourceGroup != "" && conf.IdentityPoolClaimDurationHours == 0 {
		return nil, fmt.Errorf("missing env var: PLATFORM_WORKLOAD_IDENTITY_POOL_CLAIM_DURATION_HOURS")
	}

	return &conf, nil
}

func New(log *logrus.Entry, conf *ClusterConfig) (*Cluster, error) {
	azEnvironment, err := azureclient.EnvironmentFromName(conf.AzureEnvironment)

	if err != nil {
		return nil, fmt.Errorf("can't parse Azure environment: %w", err)
	}

	options := azEnvironment.EnvironmentCredentialOptions()

	spTokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		return nil, err
	}

	spGraphClient, err := azEnvironment.NewGraphServiceClient(spTokenCredential)
	if err != nil {
		return nil, err
	}

	scopes := []string{azEnvironment.ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(spTokenCredential, scopes)

	armOption := armsdk.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: options.Cloud,
		},
	}

	clientOptions := azEnvironment.ArmClientOptions()

	vaultClient, err := armkeyvault.NewVaultsClient(conf.SubscriptionID, spTokenCredential, &armOption)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := armnetwork.NewSecurityGroupsClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	diskEncryptionSetsClient := compute.NewDiskEncryptionSetsClient(conf.SubscriptionID, authorizer)

	if err != nil {
		return nil, err
	}

	subnetsClient, err := armnetwork.NewSubnetsClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	routeTablesClient, err := armnetwork.NewRouteTablesClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	virtualNetworkPeeringsClient, err := armnetwork.NewVirtualNetworkPeeringsClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	msiClient, err := armmsiclient.NewUserAssignedIdentitiesClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	clusterClient := &internalClient[mgmtredhatopenshift20240812preview.OpenShiftCluster, v20240812preview.OpenShiftCluster]{
		externalClient: redhatopenshift20240812preview.NewOpenShiftClustersClient(&azEnvironment, conf.SubscriptionID, authorizer),
		converter:      api.APIs[v20240812preview.APIVersion].OpenShiftClusterConverter,
	}

	c := &Cluster{
		log:    log,
		Config: conf,
		//		env:                environment,
		workloadIdentities: make(map[string]api.PlatformWorkloadIdentity),

		spGraphClient:            spGraphClient,
		deployments:              features.NewDeploymentsClient(&azEnvironment, conf.SubscriptionID, authorizer),
		groups:                   features.NewResourceGroupsClient(&azEnvironment, conf.SubscriptionID, authorizer),
		openshiftclusters:        clusterClient,
		securitygroups:           securityGroupsClient,
		subnets:                  subnetsClient,
		routetables:              routeTablesClient,
		roleassignments:          authorization.NewRoleAssignmentsClient(&azEnvironment, conf.SubscriptionID, authorizer),
		roledefinitions:          authorization.NewRoleDefinitionsClient(&azEnvironment, conf.SubscriptionID, authorizer),
		peerings:                 virtualNetworkPeeringsClient,
		vaultsClient:             vaultClient,
		msiClient:                msiClient,
		diskEncryptionSetsClient: diskEncryptionSetsClient,
	}

	if c.Config.IsCI && c.Config.IsLocalDevelopmentMode() {
		// Only peer if CI=true and RP_MODE=development
		c.ciParentVnet = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vpn-vnet", c.Config.SubscriptionID, c.Config.RPResourceGroup)

		r, err := azure.ParseResourceID(c.ciParentVnet)
		if err != nil {
			return nil, err
		}

		ciVirtualNetworkPeeringsClient, err := armnetwork.NewVirtualNetworkPeeringsClient(r.SubscriptionID, spTokenCredential, clientOptions)
		if err != nil {
			return nil, err
		}

		c.ciParentVnetPeerings = ciVirtualNetworkPeeringsClient
	}

	return c, nil
}

type appDetails struct {
	applicationId     string
	applicationSecret string
	SPId              string
}

func (c *Cluster) createApp(ctx context.Context, clusterName string) (applicationDetails appDetails, err error) {
	c.log.Infof("Creating AAD application")
	appID, appSecret, err := c.createApplication(ctx, "aro-"+clusterName)
	if err != nil {
		return appDetails{}, err
	}

	c.log.Infof("Creating service principal")
	spID, err := c.createServicePrincipal(ctx, appID)
	if err != nil {
		return appDetails{}, err
	}

	return appDetails{appID, appSecret, spID}, nil
}

func (c *Cluster) SetupServicePrincipalRoleAssignments(ctx context.Context, diskEncryptionSetID string, clusterServicePrincipalID string) error {
	c.log.Info("creating role assignments")

	for _, scope := range []struct{ resource, role string }{
		{"/subscriptions/" + c.Config.SubscriptionID + "/resourceGroups/" + c.Config.VnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/dev-vnet", rbac.RoleNetworkContributor},
		{"/subscriptions/" + c.Config.SubscriptionID + "/resourceGroups/" + c.Config.VnetResourceGroup + "/providers/Microsoft.Network/routeTables/" + c.Config.ClusterName + "-rt", rbac.RoleNetworkContributor},
		{diskEncryptionSetID, rbac.RoleReader},
	} {
		for _, principalID := range []string{clusterServicePrincipalID, c.Config.FPServicePrincipalID} {
			for i := 0; i < 5; i++ {
				_, err := c.roleassignments.Create(
					ctx,
					scope.resource,
					uuid.DefaultGenerator.Generate(),
					mgmtauthorization.RoleAssignmentCreateParameters{
						RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
							RoleDefinitionID: to.StringPtr("/subscriptions/" + c.Config.SubscriptionID + "/providers/Microsoft.Authorization/roleDefinitions/" + scope.role),
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

	return nil
}

func (c *Cluster) GetPlatformWIRoles() ([]api.PlatformWorkloadIdentityRole, error) {
	var wiRoleSets []api.PlatformWorkloadIdentityRoleSetProperties

	if err := json.Unmarshal([]byte(c.Config.WorkloadIdentityRoles), &wiRoleSets); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	for _, rs := range wiRoleSets {
		if strings.HasPrefix(c.Config.OSClusterVersion, rs.OpenShiftVersion) {
			return rs.PlatformWorkloadIdentityRoles, nil
		}
	}

	return nil, fmt.Errorf("workload identity role sets for version %s not found", c.Config.OSClusterVersion)
}

func (c *Cluster) SetupWorkloadIdentity(ctx context.Context, vnetResourceGroup string) error {
	platformWorkloadIdentityRoles, err := c.GetPlatformWIRoles()
	if err != nil {
		return fmt.Errorf("failed parsing platformWI Roles: %w", err)
	}

	platformWorkloadIdentityRoles = append(platformWorkloadIdentityRoles, api.PlatformWorkloadIdentityRole{
		OperatorName:     "aro-Cluster",
		RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e",
	})

	c.log.Info("Assigning role to mock msi client")
	identityRg := vnetResourceGroup
	if c.Config.IdentityPoolResourceGroup != "" {
		identityRg = c.Config.IdentityPoolResourceGroup
	}
	c.roleassignments.Create(
		ctx,
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.Config.SubscriptionID, identityRg),
		uuid.DefaultGenerator.Generate(),
		mgmtauthorization.RoleAssignmentCreateParameters{
			RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
				RoleDefinitionID: to.StringPtr("/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e"),
				PrincipalID:      &c.Config.MockMSIObjectID,
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
	)

	newIdentities := []*armmsi.Identity{}
	if c.Config.IdentityPoolResourceGroup != "" {
		c.log.Info("Claiming workload identities")
		pool := NewManagedIdentityPool(c.msiClient, c.Config.IdentityPoolResourceGroup)
		timeout := time.Duration(c.Config.IdentityPoolClaimDurationHours) * time.Hour
		identities, err := pool.ClaimIdentities(ctx, len(platformWorkloadIdentityRoles), c.Config.VnetResourceGroup, c.Config.ClusterName, timeout)
		if err != nil {
			return err
		}
		newIdentities = append(newIdentities, identities...)
	} else {
		for _, wi := range platformWorkloadIdentityRoles {
			c.log.Infof("creating WI: %s", wi.OperatorName)
			resp, err := c.msiClient.CreateOrUpdate(ctx, vnetResourceGroup, wi.OperatorName, armmsi.Identity{
				Location: to.StringPtr(c.Config.Location),
			}, nil)
			if err != nil {
				return err
			}
			newIdentities = append(newIdentities, &resp.Identity)
		}
	}

	for i, wi := range platformWorkloadIdentityRoles {
		currentIdentity := newIdentities[i]
		_, err = c.roleassignments.Create(
			ctx,
			fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.Config.SubscriptionID, vnetResourceGroup),
			uuid.DefaultGenerator.Generate(),
			mgmtauthorization.RoleAssignmentCreateParameters{
				RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
					RoleDefinitionID: &wi.RoleDefinitionID,
					PrincipalID:      currentIdentity.Properties.PrincipalID,
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
		)
		if err != nil {
			return err
		}

		if wi.OperatorName != "aro-Cluster" {
			c.workloadIdentities[wi.OperatorName] = api.PlatformWorkloadIdentity{
				ResourceID: *currentIdentity.ID,
			}
		}
	}

	return nil
}

func (c *Cluster) Create(ctx context.Context) error {
	c.log.Info("Creating cluster")
	clusterGet, err := c.openshiftclusters.Get(ctx, c.Config.VnetResourceGroup, c.Config.ClusterName)
	c.log.Info("Got cluster ref")

	if err == nil {
		if clusterGet.Properties.ProvisioningState == api.ProvisioningStateFailed {
			return fmt.Errorf("cluster exists and is in failed provisioning state, please delete and retry: %s, %s", clusterGet.ID, c.Config.VnetResourceGroup)
		}
		c.log.Print("cluster already exists, skipping create")
		return nil
	}

	appDetails := appDetails{}
	if !c.Config.UseWorkloadIdentity {
		c.log.Info("Creating app")
		appDetails, err = c.createApp(ctx, c.Config.ClusterName)
		if err != nil {
			return err
		}
	}

	visibility := api.VisibilityPublic

	if c.Config.IsPrivate || c.Config.NoInternet {
		visibility = api.VisibilityPrivate
	}

	if c.Config.IsCI {
		c.log.Infof("creating resource group")
		_, err = c.groups.CreateOrUpdate(ctx, c.Config.VnetResourceGroup, mgmtfeatures.ResourceGroup{
			Location: to.StringPtr(c.Config.Location),
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

	addressPrefix, masterSubnet, workerSubnet, err := c.generateSubnets()

	if err != nil {
		return err
	}

	diskEncryptionSetName := fmt.Sprintf(
		"%s%s",
		c.Config.VnetResourceGroup,
		generator.SharedDiskEncryptionSetNameSuffix,
	)

	var kvName string
	if !c.Config.IsCI {
		if len(c.Config.VnetResourceGroup) > 10 {
			// keyvault names need to have a maximum length of 24,
			// so we need to cut off some chars if the resource group name is too long
			kvName = c.Config.VnetResourceGroup[:10] + generator.SharedDiskEncryptionKeyVaultNameSuffix
		} else {
			kvName = c.Config.VnetResourceGroup + generator.SharedDiskEncryptionKeyVaultNameSuffix
		}
	} else {
		// if DES already exists in RG, then reuse KV hosting the key of this DES,
		// otherwise, name is limited to 24 characters, but must be globally unique,
		// so we generate a name randomly until it is available
		diskEncryptionSet, err := c.diskEncryptionSetsClient.Get(ctx, c.Config.VnetResourceGroup, diskEncryptionSetName)
		if err == nil {
			if diskEncryptionSet.EncryptionSetProperties == nil ||
				diskEncryptionSet.EncryptionSetProperties.ActiveKey == nil ||
				diskEncryptionSet.EncryptionSetProperties.ActiveKey.SourceVault == nil ||
				diskEncryptionSet.EncryptionSetProperties.ActiveKey.SourceVault.ID == nil {
				return fmt.Errorf("no valid Key Vault found in Disk Encryption Set: %v. Delete the Disk Encryption Set and retry", diskEncryptionSet)
			}
			ID := *diskEncryptionSet.EncryptionSetProperties.ActiveKey.SourceVault.ID
			var found bool
			_, kvName, found = strings.Cut(ID, "/providers/Microsoft.KeyVault/vaults/")
			if !found {
				return fmt.Errorf("could not find Key Vault name in ID: %v", ID)
			}
		} else {
			if autorestErr, ok := err.(autorest.DetailedError); !ok ||
				autorestErr.Response == nil ||
				autorestErr.Response.StatusCode != http.StatusNotFound {
				return fmt.Errorf("failed to get Disk Encryption Set: %v", err)
			}
			for {
				kvName = "kv-" + uuid.DefaultGenerator.Generate()[:21]
				result, err := c.vaultsClient.CheckNameAvailability(
					ctx,
					sdkkeyvault.VaultCheckNameAvailabilityParameters{Name: &kvName, Type: to.StringPtr("Microsoft.KeyVault/vaults")},
					nil,
				)
				if err != nil {
					return err
				}

				if result.NameAvailable == nil {
					return fmt.Errorf("have unexpected nil NameAvailable for key vault: %v", kvName)
				}

				if *result.NameAvailable {
					break
				}
				c.log.Infof("key vault %v is not available and we will try an other one", kvName)
			}
		}
	}

	parameters := map[string]*arm.ParametersParameter{
		"clusterName":         {Value: c.Config.ClusterName},
		"ci":                  {Value: c.Config.IsCI},
		"vnetAddressPrefix":   {Value: addressPrefix},
		"masterAddressPrefix": {Value: masterSubnet},
		"workerAddressPrefix": {Value: workerSubnet},
		"kvName":              {Value: kvName},
	}

	// TODO: ick
	if os.Getenv("NO_INTERNET") != "" {
		parameters["routes"] = &arm.ParametersParameter{
			Value: []sdknetwork.Route{
				{
					Properties: &sdknetwork.RoutePropertiesFormat{
						AddressPrefix: pointerutils.ToPtr("0.0.0.0/0"),
						NextHopType:   pointerutils.ToPtr(sdknetwork.RouteNextHopTypeNone),
					},
					Name: pointerutils.ToPtr("blackhole"),
				},
			},
		}
	}

	armctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	c.log.Info("predeploying ARM template")
	err = c.deployments.CreateOrUpdateAndWait(armctx, c.Config.VnetResourceGroup, c.Config.ClusterName, mgmtfeatures.Deployment{
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
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/diskEncryptionSets/%s",
		c.Config.SubscriptionID,
		c.Config.VnetResourceGroup,
		diskEncryptionSetName,
	)

	if c.Config.UseWorkloadIdentity {
		c.log.Info("creating WIs")
		if err := c.SetupWorkloadIdentity(ctx, c.Config.VnetResourceGroup); err != nil {
			return fmt.Errorf("error setting up Workload Identity Roles: %w", err)
		}
	} else {
		c.log.Info("creating Classic role assignments")
		c.SetupServicePrincipalRoleAssignments(ctx, diskEncryptionSetID, appDetails.SPId)
	}
	fipsMode := true

	// Don't install with FIPS in a local dev, non-CI environment
	if !c.Config.IsCI && c.Config.IsLocalDevelopmentMode() {
		fipsMode = false
	}

	c.log.Info("creating cluster")
	err = c.createCluster(ctx, c.Config.VnetResourceGroup, c.Config.ClusterName, appDetails.applicationId, appDetails.applicationSecret, diskEncryptionSetID, visibility, c.Config.OSClusterVersion, fipsMode)

	if err != nil {
		return err
	}

	if c.Config.IsCI {
		c.log.Info("fixing up NSGs")
		err = c.fixupNSGs(ctx, c.Config.VnetResourceGroup, c.Config.ClusterName)
		if err != nil {
			return err
		}

		if env.IsLocalDevelopmentMode() {
			c.log.Info("peering subnets to CI infra")
			err = c.peerSubnetsToCI(ctx, c.Config.VnetResourceGroup)
			if err != nil {
				return err
			}
		}
	}

	c.log.Info("done")
	return nil
}

// ipRangesContainCIDR checks, weather any of the ipRanges overlap with the cidr string. In case cidr isn't valid, false is returned.
func ipRangesContainCIDR(ipRanges []*net.IPNet, cidr string) (bool, error) {
	_, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}

	for _, snet := range ipRanges {
		if snet.Contains(cidrNet.IP) || cidrNet.Contains(snet.IP) {
			return true, nil
		}
	}
	return false, nil
}

// GetIPRangesFromSubnet converts a given azure subnet to a list if IPNets.
// Because an az subnet can cover multiple ipranges, we need to return a slice
// instead of just a single ip range. This function never errors. If something
// goes wrong, it instead returns an empty list.
func GetIPRangesFromSubnet(subnet *sdknetwork.Subnet) []*net.IPNet {
	ipRanges := []*net.IPNet{}
	if subnet.Properties.AddressPrefix != nil {
		_, ipRange, err := net.ParseCIDR(*subnet.Properties.AddressPrefix)
		if err == nil {
			ipRanges = append(ipRanges, ipRange)
		}
	}

	if subnet.Properties.AddressPrefixes == nil {
		return ipRanges
	}

	for _, snetPrefix := range subnet.Properties.AddressPrefixes {
		_, ipRange, err := net.ParseCIDR(*snetPrefix)
		if err == nil {
			ipRanges = append(ipRanges, ipRange)
		}
	}
	return ipRanges
}

func (c *Cluster) generateSubnets() (vnetPrefix string, masterSubnet string, workerSubnet string, err error) {
	// pick a random 23 in range [10.3.0.0, 10.127.255.0], making sure it doesn't
	// conflict with other subnets present in out dev-vnet
	// 10.0.0.0/16 is used by dev-vnet to host CI
	// 10.1.0.0/24 is used by rp-vnet to host Proxy VM
	// 10.2.0.0/24 is used by dev-vpn-vnet to host VirtualNetworkGateway

	allSubnets, err := c.subnets.List(context.Background(), c.Config.VnetResourceGroup, "dev-vnet", nil)
	if err != nil {
		c.log.Warnf("Error getting existing subnets. Continuing regardless: %v", err)
	}

	ipRanges := []*net.IPNet{}
	for _, snet := range allSubnets {
		ipRanges = append(ipRanges, GetIPRangesFromSubnet(snet)...)
	}

	for i := 1; i < GenerateSubnetMaxTries; i++ {
		var x, y int
		// Local Dev clusters are limited to /16 dev-vnet
		if !c.Config.IsCI {
			x, y = 0, 2*rand.Intn(128)
		} else {
			x, y = rand.Intn((124))+3, 2*rand.Intn(128)
		}
		c.log.Infof("Generate Subnet try: %d\n", i)
		vnetPrefix = fmt.Sprintf("10.%d.%d.0/23", x, y)
		masterSubnet = fmt.Sprintf("10.%d.%d.0/24", x, y)
		workerSubnet = fmt.Sprintf("10.%d.%d.0/24", x, y+1)

		masterSubnetOverlaps, err := ipRangesContainCIDR(ipRanges, masterSubnet)
		if err != nil || masterSubnetOverlaps {
			continue
		}

		workerSubnetOverlaps, err := ipRangesContainCIDR(ipRanges, workerSubnet)
		if err != nil || workerSubnetOverlaps {
			continue
		}

		c.log.Infof("Generated subnets: vnet: %s, master: %s, worker: %s\n", vnetPrefix, masterSubnet, workerSubnet)
		return vnetPrefix, masterSubnet, workerSubnet, nil
	}

	return vnetPrefix, masterSubnet, workerSubnet, fmt.Errorf("was not able to generate master and worker subnets after %v tries", GenerateSubnetMaxTries)
}

func (c *Cluster) Delete(ctx context.Context, vnetResourceGroup, clusterName string) error {
	c.log.Infof("Deleting cluster %s in resource group %s", clusterName, vnetResourceGroup)
	var errs []error

	if c.Config.IsCI {
		oc, err := c.openshiftclusters.Get(ctx, vnetResourceGroup, clusterName)
		clusterResourceGroup := fmt.Sprintf("aro-%s", clusterName)
		if err != nil {
			c.log.Errorf("CI E2E cluster %s not found in resource group %s", clusterName, vnetResourceGroup)
			errs = append(errs, err)
		}
		if oc.Properties.ServicePrincipalProfile != nil {
			errs = append(errs,
				c.deleteApplication(ctx, oc.Properties.ServicePrincipalProfile.ClientID),
			)
		}

		errs = append(errs,
			c.deleteCluster(ctx, vnetResourceGroup, clusterName),
			c.deleteWimiRoleAssignments(ctx, vnetResourceGroup),
			c.deleteWI(ctx, vnetResourceGroup),
			c.ensureResourceGroupDeleted(ctx, clusterResourceGroup),
			c.deleteResourceGroup(ctx, vnetResourceGroup),
		)

		if env.IsLocalDevelopmentMode() { //PR E2E
			errs = append(errs,
				c.deleteVnetPeerings(ctx, vnetResourceGroup),
			)
		}
	} else {
		errs = append(errs,
			c.deleteRoleAssignments(ctx, vnetResourceGroup, clusterName),
			c.deleteCluster(ctx, vnetResourceGroup, clusterName),
			c.deleteWimiRoleAssignments(ctx, vnetResourceGroup),
			c.deleteWI(ctx, vnetResourceGroup),
			c.deleteDeployment(ctx, vnetResourceGroup, clusterName), // Deleting the deployment does not clean up the associated resources
			c.deleteVnetResources(ctx, vnetResourceGroup, "dev-vnet", clusterName),
		)
	}

	c.log.Info("done")
	return errors.Join(errs...)
}

func (c *Cluster) deleteWI(ctx context.Context, resourceGroup string) error {
	if !c.Config.UseWorkloadIdentity {
		c.log.Info("Skipping deletion of workload identity roles")
		return nil
	}

	if c.Config.IdentityPoolResourceGroup != "" {
		c.log.Infof("Freeing claimed Workload Identities in RG %s", c.Config.IdentityPoolResourceGroup)
		pool := NewManagedIdentityPool(c.msiClient, c.Config.IdentityPoolResourceGroup)
		return pool.FreeAllIdentitiesOfCluster(ctx, c.Config.VnetResourceGroup, c.Config.ClusterName)
	}

	c.log.Info("deleting WIs")
	platformWorkloadIdentityRoles, err := c.GetPlatformWIRoles()
	if err != nil {
		return fmt.Errorf("failure parsing Platform WI Roles, unable to remove them: %w", err)
	}
	platformWorkloadIdentityRoles = append(platformWorkloadIdentityRoles, api.PlatformWorkloadIdentityRole{
		OperatorName:     "aro-Cluster",
		RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e",
	})
	for _, wi := range platformWorkloadIdentityRoles {
		c.log.Infof("deleting WI: %s", wi.OperatorName)
		_, err := c.msiClient.Delete(ctx, resourceGroup, wi.OperatorName, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// createCluster created new clusters, based on where it is running.
// development - using preview api
// production - using stable GA api
func (c *Cluster) createCluster(ctx context.Context, vnetResourceGroup, clusterName, clientID, clientSecret, diskEncryptionSetID string, visibility api.Visibility, osClusterVersion string, fipsEnabled bool) error {
	fipsMode := api.FipsValidatedModulesDisabled
	if fipsEnabled {
		fipsMode = api.FipsValidatedModulesEnabled
	}

	// using internal representation for "singe source" of options
	oc := api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain:               strings.ToLower(clusterName),
				ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.Config.SubscriptionID, "aro-"+clusterName),
				FipsValidatedModules: fipsMode,
				Version:              osClusterVersion,
				PullSecret:           api.SecureString(os.Getenv("USER_PULL_SECRET")),
			},
			NetworkProfile: api.NetworkProfile{
				PodCIDR:                "10.128.0.0/14",
				ServiceCIDR:            "172.30.0.0/16",
				SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
			},
			MasterProfile: api.MasterProfile{
				VMSize:              api.VMSize(c.Config.MasterVMSize),
				SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.Config.SubscriptionID, vnetResourceGroup, clusterName),
				EncryptionAtHost:    api.EncryptionAtHostEnabled,
				DiskEncryptionSetID: diskEncryptionSetID,
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:                "worker",
					VMSize:              api.VMSize(c.Config.WorkerVMSize),
					DiskSizeGB:          128,
					SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-worker", c.Config.SubscriptionID, vnetResourceGroup, clusterName),
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
		Location: c.Config.Location,
	}

	if c.Config.UseWorkloadIdentity {
		oc.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{
			PlatformWorkloadIdentities: c.workloadIdentities,
		}

		oc.Identity = &api.ManagedServiceIdentity{
			Type:     api.ManagedServiceIdentityUserAssigned,
			TenantID: c.Config.TenantID,
			UserAssignedIdentities: map[string]api.UserAssignedIdentity{
				fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", c.Config.SubscriptionID, vnetResourceGroup, "aro-Cluster"): {},
			},
		}
	} else {
		if clientID != "" && clientSecret != "" {
			oc.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
				ClientID:     clientID,
				ClientSecret: api.SecureString(clientSecret),
			}
		}
	}

	if c.Config.IsLocalDevelopmentMode() {
		err := c.registerSubscription()
		if err != nil {
			return err
		}

		err = c.ensureDefaultVersionInCosmosdb(ctx)
		if err != nil {
			return err
		}
		// If we're in local dev mode and the user has not overridden the default VM size, use a smaller size for cost-saving purposes
		if c.Config.WorkerVMSize == DefaultWorkerVmSize.String() {
			oc.Properties.WorkerProfiles[0].VMSize = api.VMSizeStandardD2sV3
		}
	}

	return c.openshiftclusters.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, &oc)
}

func (c *Cluster) registerSubscription() error {
	b, err := json.Marshal(&api.Subscription{
		State: api.SubscriptionStateRegistered,
		Properties: &api.SubscriptionProperties{
			TenantID: c.Config.TenantID,
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

	req, err := http.NewRequest(http.MethodPut, localDefaultURL+"/subscriptions/"+c.Config.SubscriptionID+"?api-version=2.0", bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := insecureLocalClient().Do(req)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// getVersionsInCosmosDB connects to the local RP endpoint and queries the
// available OpenShiftVersions
func getVersionsInCosmosDB(ctx context.Context) ([]*api.OpenShiftVersion, error) {
	type getVersionResponse struct {
		Value []*api.OpenShiftVersion `json:"value"`
	}

	getRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, localDefaultURL+"/admin/versions", &bytes.Buffer{})
	if err != nil {
		return nil, fmt.Errorf("error creating get versions request: %w", err)
	}

	getRequest.Header.Set("Content-Type", "application/json")

	getResponse, err := insecureLocalClient().Do(getRequest)
	if err != nil {
		return nil, fmt.Errorf("error couldn't retrieve versions in cosmos db: %w", err)
	}

	parsedResponse := getVersionResponse{}
	decoder := json.NewDecoder(getResponse.Body)
	err = decoder.Decode(&parsedResponse)

	return parsedResponse.Value, err
}

// ensureDefaultVersionInCosmosdb puts a default openshiftversion into the
// cosmos DB IF it doesn't already contain an entry for the default version. It
// is hardcoded to use the local-RP endpoint
//
// It returns without an error when a default version is already present or a
// default version was successfully put into the db.
func (c *Cluster) ensureDefaultVersionInCosmosdb(ctx context.Context) error {
	versionsInDB, err := getVersionsInCosmosDB(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query versions in cosmosdb: %w", err)
	}

	for _, versionFromDB := range versionsInDB {
		if versionFromDB.Properties.Version == version.DefaultInstallStream.Version.String() {
			c.log.Debugf("Version %s already in DB. Not overwriting existing one.", version.DefaultInstallStream.Version.String())
			return nil
		}
	}

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

	req, err := http.NewRequest(http.MethodPut, localDefaultURL+"/admin/versions", bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := insecureLocalClient().Do(req)
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
	var nsgs []*sdknetwork.SecurityGroup
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var err error
		nsgs, err = c.securitygroups.List(ctx, "aro-"+clusterName, nil)
		return len(nsgs) > 0, err
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	for _, subnetName := range []string{clusterName + "-master", clusterName + "-worker"} {
		resp, err := c.subnets.Get(ctx, vnetResourceGroup, "dev-vnet", subnetName, nil)
		if err != nil {
			return err
		}
		subnet := resp.Subnet

		subnet.Properties.NetworkSecurityGroup = &sdknetwork.SecurityGroup{
			ID: nsgs[0].ID,
		}

		err = c.subnets.CreateOrUpdateAndWait(ctx, vnetResourceGroup, "dev-vnet", subnetName, subnet, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) deleteRoleAssignments(ctx context.Context, vnetResourceGroup, clusterName string) error {
	if c.Config.UseWorkloadIdentity {
		c.log.Print("Skipping deletion of service principal role assignments")
	}
	c.log.Print("deleting role assignments")
	oc, err := c.openshiftclusters.Get(ctx, vnetResourceGroup, clusterName)
	if err != nil {
		return fmt.Errorf("error getting cluster document: %w", err)
	}
	spObjID, err := utilgraph.GetServicePrincipalIDByAppID(ctx, c.spGraphClient, oc.Properties.ServicePrincipalProfile.ClientID)
	if err != nil {
		return fmt.Errorf("error getting service principal for cluster: %w", err)
	}
	if spObjID == nil {
		return nil
	}

	roleAssignments, err := c.roleassignments.ListForResourceGroup(ctx, vnetResourceGroup, fmt.Sprintf("principalId eq '%s'", *spObjID))
	if err != nil {
		return fmt.Errorf("error listing role assignments for service principal: %w", err)
	}

	for _, roleAssignment := range roleAssignments {
		if strings.HasPrefix(
			strings.ToLower(*roleAssignment.Scope),
			strings.ToLower("/subscriptions/"+c.Config.SubscriptionID+"/resourceGroups/"+vnetResourceGroup),
		) {
			// Don't delete inherited role assignments, only those resource group level or below
			c.log.Infof("deleting role assignment %s", *roleAssignment.Name)
			_, err = c.roleassignments.Delete(ctx, *roleAssignment.Scope, *roleAssignment.Name)
			if err != nil {
				return fmt.Errorf("error deleting role assignment %s: %w", *roleAssignment.Name, err)
			}
		}
	}

	return nil
}

func (c *Cluster) deleteWimiRoleAssignments(ctx context.Context, vnetResourceGroup string) error {
	if !c.Config.UseWorkloadIdentity {
		c.log.Print("Skipping deletion of wimi roleassignments")
		return nil
	}
	c.log.Print("deleting wimi role assignments")

	var wiRoleSets []api.PlatformWorkloadIdentityRoleSetProperties
	if err := json.Unmarshal([]byte(c.Config.WorkloadIdentityRoles), &wiRoleSets); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	platformWorkloadIdentityRoles := append(wiRoleSets[0].PlatformWorkloadIdentityRoles, api.PlatformWorkloadIdentityRole{
		OperatorName:     "aro-Cluster",
		RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e",
	})
	for _, wi := range platformWorkloadIdentityRoles {
		resp, err := c.msiClient.Get(ctx, vnetResourceGroup, wi.OperatorName, nil)
		if err != nil {
			return err
		}
		roleAssignments, err := c.roleassignments.ListForResourceGroup(ctx, vnetResourceGroup, fmt.Sprintf("principalId eq '%s'", *resp.Properties.PrincipalID))
		if err != nil {
			return fmt.Errorf("error listing role assignments for service principal: %w", err)
		}
		for _, roleAssignment := range roleAssignments {
			if strings.HasPrefix(
				strings.ToLower(*roleAssignment.Scope),
				strings.ToLower("/subscriptions/"+c.Config.SubscriptionID+"/resourceGroups/"+vnetResourceGroup),
			) {
				// Don't delete inherited role assignments, only those resource group level or below
				c.log.Infof("deleting role assignment %s", *roleAssignment.Name)
				_, err = c.roleassignments.Delete(ctx, *roleAssignment.Scope, *roleAssignment.Name)
				if err != nil {
					return fmt.Errorf("error deleting role assignment %s: %w", *roleAssignment.Name, err)
				}
			}
		}
	}
	return nil
}

func (c *Cluster) deleteCluster(ctx context.Context, resourceGroup, clusterName string) error {
	c.log.Printf("deleting cluster %s", clusterName)
	if err := c.openshiftclusters.DeleteAndWait(ctx, resourceGroup, clusterName); err != nil {
		return fmt.Errorf("error deleting cluster %s: %w", clusterName, err)
	}
	return nil
}

func (c *Cluster) ensureResourceGroupDeleted(ctx context.Context, resourceGroupName string) error {
	c.log.Printf("deleting resource group %s", resourceGroupName)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return wait.PollImmediateUntil(5*time.Second, func() (bool, error) {
		_, err := c.groups.Get(ctx, resourceGroupName)
		if azureerrors.ResourceGroupNotFound(err) {
			c.log.Infof("finished deleting resource group %s", resourceGroupName)
			return true, nil
		}
		return false, fmt.Errorf("failed to delete resource group %s with %s", resourceGroupName, err)
	}, timeoutCtx.Done())
}

func (c *Cluster) deleteResourceGroup(ctx context.Context, resourceGroup string) error {
	c.log.Printf("deleting resource group %s", resourceGroup)
	if _, err := c.groups.Get(ctx, resourceGroup); err != nil {
		c.log.Printf("error getting resource group %s, skipping deletion: %v", resourceGroup, err)
		return nil
	}

	if err := c.groups.DeleteAndWait(ctx, resourceGroup); err != nil {
		return fmt.Errorf("error deleting resource group: %w", err)
	}

	return nil
}

func (c *Cluster) deleteVnetPeerings(ctx context.Context, resourceGroup string) error {
	r, err := azure.ParseResourceID(c.ciParentVnet)
	if err == nil {
		err = c.ciParentVnetPeerings.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, resourceGroup+"-peer", nil)
	}
	if err != nil {
		return fmt.Errorf("error deleting vnet peerings: %w", err)
	}

	return nil
}

func (c *Cluster) deleteDeployment(ctx context.Context, resourceGroup, clusterName string) error {
	c.log.Info("deleting deployment")
	if err := c.deployments.DeleteAndWait(ctx, resourceGroup, clusterName); err != nil {
		return fmt.Errorf("error deleting deployment: %w", err)
	}
	return nil
}

func (c *Cluster) deleteVnetResources(ctx context.Context, resourceGroup, vnetName, clusterName string) error {
	var errs []error

	c.log.Info("deleting master/worker subnets")
	if err := c.subnets.DeleteAndWait(ctx, resourceGroup, vnetName, clusterName+"-master", nil); err != nil {
		c.log.Errorf("error when deleting master subnet: %v", err)
		errs = append(errs, err)
	}

	if err := c.subnets.DeleteAndWait(ctx, resourceGroup, vnetName, clusterName+"-worker", nil); err != nil {
		c.log.Errorf("error when deleting worker subnet: %v", err)
		errs = append(errs, err)
	}

	c.log.Info("deleting route table")
	if err := c.routetables.DeleteAndWait(ctx, resourceGroup, clusterName+"-rt", nil); err != nil {
		c.log.Errorf("error when deleting route table: %v", err)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *Cluster) peerSubnetsToCI(ctx context.Context, vnetResourceGroup string) error {
	cluster := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet", c.Config.SubscriptionID, vnetResourceGroup)

	r, err := azure.ParseResourceID(c.ciParentVnet)
	if err != nil {
		return err
	}

	clusterProp := &sdknetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &sdknetwork.SubResource{
			ID: &c.ciParentVnet,
		},
		AllowVirtualNetworkAccess: to.BoolPtr(true),
		AllowForwardedTraffic:     to.BoolPtr(true),
		UseRemoteGateways:         to.BoolPtr(true),
	}
	rpProp := &sdknetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &sdknetwork.SubResource{
			ID: &cluster,
		},
		AllowVirtualNetworkAccess: to.BoolPtr(true),
		AllowForwardedTraffic:     to.BoolPtr(true),
		AllowGatewayTransit:       to.BoolPtr(true),
	}

	err = c.peerings.CreateOrUpdateAndWait(ctx, vnetResourceGroup, "dev-vnet", r.ResourceGroup+"-peer", sdknetwork.VirtualNetworkPeering{Properties: clusterProp}, nil)
	if err != nil {
		return err
	}

	err = c.ciParentVnetPeerings.CreateOrUpdateAndWait(ctx, r.ResourceGroup, r.ResourceName, vnetResourceGroup+"-peer", sdknetwork.VirtualNetworkPeering{Properties: rpProp}, nil)
	if err != nil {
		return err
	}

	return err
}
