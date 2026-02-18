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

	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"k8s.io/apimachinery/pkg/util/wait"

	armsdk "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	sdkkeyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	v20250725 "github.com/Azure/ARO-RP/pkg/api/v20250725"
	mgmtredhatopenshift20250725 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2025-07-25/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/deploy/assets"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armkeyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	redhatopenshift20250725 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2025-07-25/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/pkg/util/vms"
)

type ClusterConfig struct {
	ClusterName           string `mapstructure:"CLUSTER"`
	SubscriptionID        string `mapstructure:"AZURE_SUBSCRIPTION_ID"`
	TenantID              string `mapstructure:"AZURE_TENANT_ID"`
	Location              string `mapstructure:"LOCATION"`
	AzureEnvironment      string `mapstructure:"AZURE_ENVIRONMENT"`
	UseWorkloadIdentity   bool   `mapstructure:"USE_WI"`
	WorkloadIdentityRoles string `mapstructure:"PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"`
	IsCI                  bool   `mapstructure:"CI"`
	RpMode                string `mapstructure:"RP_MODE"`
	VnetResourceGroup     string `mapstructure:"CLUSTER_RESOURCEGROUP"`
	RPResourceGroup       string `mapstructure:"RESOURCEGROUP"`
	OSClusterVersion      string `mapstructure:"OS_CLUSTER_VERSION"`
	FPServicePrincipalID  string `mapstructure:"AZURE_FP_SERVICE_PRINCIPAL_ID"`
	IsPrivate             bool   `mapstructure:"PRIVATE_CLUSTER"`
	NoInternet            bool   `mapstructure:"NO_INTERNET"`
	MockMSIObjectID       string `mapstructure:"MOCK_MSI_OBJECT_ID"`

	MasterVMSize vms.VMSize `mapstructure:"MASTER_VM_SIZE"`
	WorkerVMSize vms.VMSize `mapstructure:"WORKER_VM_SIZE"`
	// TODO: MAITIU - Do we need to touch this?
	CandidateMasterVMSizes []vms.VMSize `mapstructure:"MASTER_VM_SIZES"`
	CandidateWorkerVMSizes []vms.VMSize `mapstructure:"WORKER_VM_SIZES"`
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
	msiClient                armmsi.UserAssignedIdentitiesClient
	diskEncryptionSetsClient compute.DiskEncryptionSetsClient
}

const (
	GenerateSubnetMaxTries        = 100
	localDefaultURL        string = "https://localhost:8443"
)

func insecureLocalClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// NewClusterConfigFromEnv should only be used in the context of CI or local
// development.
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

	// Set VM size defaults only if user hasn't provided any values
	if len(conf.CandidateMasterVMSizes) == 0 {
		if conf.MasterVMSize == "" {
			conf.CandidateMasterVMSizes = vms.GetCICandidateMasterVMSizes()
		} else {
			conf.CandidateMasterVMSizes = []vms.VMSize{conf.MasterVMSize}
		}
	}
	if len(conf.CandidateWorkerVMSizes) == 0 {
		if conf.WorkerVMSize == "" {
			conf.CandidateWorkerVMSizes = vms.GetCICandidateWorkerVMSizes()
		} else {
			conf.CandidateWorkerVMSizes = []vms.VMSize{conf.WorkerVMSize}
		}
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

	msiClient, err := armmsi.NewUserAssignedIdentitiesClient(conf.SubscriptionID, spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	clusterClient := &internalClient[mgmtredhatopenshift20250725.OpenShiftCluster, v20250725.OpenShiftCluster]{
		externalClient: redhatopenshift20250725.NewOpenShiftClustersClient(&azEnvironment, conf.SubscriptionID, authorizer),
		converter:      api.APIs[v20250725.APIVersion].OpenShiftClusterConverter,
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
		msiClient:                *msiClient,
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

func (c *Cluster) SetupServicePrincipalRoleAssignments(ctx context.Context, diskEncryptionSetID string, principalIDs []string) error {
	c.log.Info("creating role assignments")

	for _, scope := range []struct{ resource, role string }{
		{"/subscriptions/" + c.Config.SubscriptionID + "/resourceGroups/" + c.Config.VnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/dev-vnet", rbac.RoleNetworkContributor},
		{"/subscriptions/" + c.Config.SubscriptionID + "/resourceGroups/" + c.Config.VnetResourceGroup + "/providers/Microsoft.Network/routeTables/" + c.Config.ClusterName + "-rt", rbac.RoleNetworkContributor},
		{diskEncryptionSetID, rbac.RoleReader},
	} {
		for _, principalID := range principalIDs {
			for i := 0; i < 5; i++ {
				_, err := c.roleassignments.Create(
					ctx,
					scope.resource,
					uuid.DefaultGenerator.Generate(),
					mgmtauthorization.RoleAssignmentCreateParameters{
						RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
							RoleDefinitionID: pointerutils.ToPtr("/subscriptions/" + c.Config.SubscriptionID + "/providers/Microsoft.Authorization/roleDefinitions/" + scope.role),
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
	c.roleassignments.Create(
		ctx,
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.Config.SubscriptionID, vnetResourceGroup),
		uuid.DefaultGenerator.Generate(),
		mgmtauthorization.RoleAssignmentCreateParameters{
			RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
				RoleDefinitionID: pointerutils.ToPtr("/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e"),
				PrincipalID:      &c.Config.MockMSIObjectID,
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
	)

	for _, wi := range platformWorkloadIdentityRoles {
		c.log.Infof("creating WI: %s", wi.OperatorName)
		resp, err := c.msiClient.CreateOrUpdate(ctx, vnetResourceGroup, wi.OperatorName, armmsi.Identity{
			Location: pointerutils.ToPtr(c.Config.Location),
		}, nil)
		if err != nil {
			return err
		}
		_, err = c.roleassignments.Create(
			ctx,
			fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.Config.SubscriptionID, vnetResourceGroup),
			uuid.DefaultGenerator.Generate(),
			mgmtauthorization.RoleAssignmentCreateParameters{
				RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
					RoleDefinitionID: &wi.RoleDefinitionID,
					PrincipalID:      resp.Properties.PrincipalID,
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
		)
		if err != nil {
			return err
		}

		if wi.OperatorName != "aro-Cluster" {
			c.workloadIdentities[wi.OperatorName] = api.PlatformWorkloadIdentity{
				ResourceID: *resp.ID,
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
			Location: pointerutils.ToPtr(c.Config.Location),
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
				diskEncryptionSet.ActiveKey == nil ||
				diskEncryptionSet.ActiveKey.SourceVault == nil ||
				diskEncryptionSet.ActiveKey.SourceVault.ID == nil {
				return fmt.Errorf("no valid Key Vault found in Disk Encryption Set: %v. Delete the Disk Encryption Set and retry", diskEncryptionSet)
			}
			ID := *diskEncryptionSet.ActiveKey.SourceVault.ID
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
					sdkkeyvault.VaultCheckNameAvailabilityParameters{Name: &kvName, Type: pointerutils.ToPtr("Microsoft.KeyVault/vaults")},
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
	}

	principalIds := []string{
		c.Config.FPServicePrincipalID,
	}

	if !c.Config.UseWorkloadIdentity {
		c.log.Info("creating cluster service principal and FPSP role assignments")
		principalIds = append(principalIds, appDetails.SPId)
	} else {
		c.log.Info("creating FPSP role assignments")
	}

	err = c.SetupServicePrincipalRoleAssignments(ctx, diskEncryptionSetID, principalIds)
	if err != nil {
		return err
	}

	fipsMode := c.Config.IsCI || !c.Config.IsLocalDevelopmentMode()

	// Don't install with FIPS in a local dev, non-CI environment

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
	c.log.Infof("Starting to delete cluster %s in resource group %s", clusterName, vnetResourceGroup)
	var errs []error

	if c.Config.IsCI {
		oc, err := c.openshiftclusters.Get(ctx, vnetResourceGroup, clusterName)
		clusterResourceGroup := fmt.Sprintf("aro-%s", clusterName)
		if err != nil {
			if azureerrors.IsNotFoundError(err) {
				c.log.Infof("Cluster %s not found in resource group %s, assuming already deleted", clusterName, vnetResourceGroup)
			} else {
				c.log.Errorf("Failed to get cluster %s in resource group %s: %v", clusterName, vnetResourceGroup, err)
				errs = append(errs, fmt.Errorf("failed to get cluster: %w", err))
			}
		}
		if oc != nil {
			if oc.Properties.ServicePrincipalProfile != nil {
				if err := c.deleteApplication(ctx, oc.Properties.ServicePrincipalProfile.ClientID); err != nil {
					c.log.Errorf("Failed to delete application: %v", err)
					errs = append(errs, fmt.Errorf("failed to delete application: %w", err))
				}
			}
		}

		if err := c.deleteCluster(ctx, vnetResourceGroup, clusterName); err != nil {
			c.log.Errorf("Failed to delete cluster: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete cluster: %w", err))
		}

		if err := c.deleteWimiRoleAssignments(ctx, vnetResourceGroup); err != nil {
			c.log.Errorf("Failed to delete workload identity role assignments: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete workload identity role assignments: %w", err))
		}

		if err := c.deleteWI(ctx, vnetResourceGroup); err != nil {
			c.log.Errorf("Failed to delete workload identities: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete workload identities: %w", err))
		}

		if err := c.checkResourceGroupDeleted(ctx, clusterResourceGroup); err != nil {
			c.log.Errorf("Failed to check resource group %s deleted: %v", clusterResourceGroup, err)
			errs = append(errs, fmt.Errorf("failed to check resource group %s deleted: %w", clusterResourceGroup, err))
		}

		if err := c.deleteResourceGroup(ctx, vnetResourceGroup); err != nil {
			c.log.Errorf("Failed to delete resource group %s: %v", vnetResourceGroup, err)
			errs = append(errs, fmt.Errorf("failed to delete resource group %s: %w", vnetResourceGroup, err))
		}

		if env.IsLocalDevelopmentMode() { // PR E2E
			if err := c.deleteVnetPeerings(ctx, vnetResourceGroup); err != nil {
				c.log.Errorf("Failed to delete VNet peerings: %v", err)
				errs = append(errs, fmt.Errorf("failed to delete VNet peerings: %w", err))
			}
		}
	} else {
		if err := c.deleteRoleAssignments(ctx, vnetResourceGroup, clusterName); err != nil {
			c.log.Errorf("Failed to delete role assignments: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete role assignments: %w", err))
		}

		if err := c.deleteCluster(ctx, vnetResourceGroup, clusterName); err != nil {
			c.log.Errorf("Failed to delete cluster: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete cluster: %w", err))
		}

		if err := c.deleteWimiRoleAssignments(ctx, vnetResourceGroup); err != nil {
			c.log.Errorf("Failed to delete workload identity role assignments: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete workload identity role assignments: %w", err))
		}

		if err := c.deleteWI(ctx, vnetResourceGroup); err != nil {
			c.log.Errorf("Failed to delete workload identities: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete workload identities: %w", err))
		}

		if err := c.deleteDeployment(ctx, vnetResourceGroup, clusterName); err != nil {
			c.log.Errorf("Failed to delete deployment: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete deployment: %w", err))
		}

		if err := c.deleteVnetResources(ctx, vnetResourceGroup, "dev-vnet", clusterName); err != nil {
			c.log.Errorf("Failed to delete VNet resources: %v", err)
			errs = append(errs, fmt.Errorf("failed to delete VNet resources: %w", err))
		}
	}

	if len(errs) > 0 {
		c.log.Errorf("Delete failed with %d error(s)", len(errs))
	} else {
		c.log.Info("Delete completed successfully")
	}
	return errors.Join(errs...)
}

func (c *Cluster) deleteWI(ctx context.Context, resourceGroup string) error {
	if !c.Config.UseWorkloadIdentity {
		c.log.Info("Skipping deletion of workload identity roles")
		return nil
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
				SubnetID:            fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/%s-master", c.Config.SubscriptionID, vnetResourceGroup, clusterName),
				EncryptionAtHost:    api.EncryptionAtHostEnabled,
				DiskEncryptionSetID: diskEncryptionSetID,
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:                "worker",
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

		if c.Config.UseWorkloadIdentity {
			err = c.ensureDefaultRoleSetInCosmosdb(ctx)
			if err != nil {
				return err
			}
		}
	}

	masterIdx := 0
	workerIdx := 0
	var err error

	for {
		if err != nil {
			// If we've already tried and failed to create the cluster, delete
			// it before retrying. Deleting first ensures that the final failed
			// cluster remains for diagnostic purposes.
			err = c.openshiftclusters.DeleteAndWait(ctx, vnetResourceGroup, clusterName)
			if err != nil {
				return fmt.Errorf("error deleting cluster after failed creation: %w", err)
			}
		}

		oc.Properties.MasterProfile.VMSize = vms.VMSize(c.Config.CandidateMasterVMSizes[masterIdx])
		oc.Properties.WorkerProfiles[0].VMSize = vms.VMSize(c.Config.CandidateWorkerVMSizes[workerIdx])
		c.log.Infof("Creating cluster %s with master VM size %s and worker VM size %s",
			clusterName, oc.Properties.MasterProfile.VMSize, oc.Properties.WorkerProfiles[0].VMSize)
		err = c.openshiftclusters.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, &oc)
		if err == nil {
			break
		}

		// Check if this is a VM SKU availability error and determine which profile failed
		isVMError, profile := azureerrors.IsVMSKUError(err)
		if !isVMError {
			return err
		}

		switch profile {
		case azureerrors.VMProfileWorker:
			c.log.WithError(err).Errorf("error creating cluster with worker VM size %s, trying next size", oc.Properties.WorkerProfiles[0].VMSize)
			workerIdx++
			if workerIdx >= len(c.Config.CandidateWorkerVMSizes) {
				return fmt.Errorf("exhausted all worker VM sizes: %w", err)
			}
		case azureerrors.VMProfileMaster:
			c.log.WithError(err).Errorf("error creating cluster with master VM size %s, trying next size", oc.Properties.MasterProfile.VMSize)
			masterIdx++
			if masterIdx >= len(c.Config.CandidateMasterVMSizes) {
				return fmt.Errorf("exhausted all master VM sizes: %w", err)
			}
		default:
			// VM size error but can't determine which profile - try next worker size first
			// (more commonly the issue in local dev mode), then cycle through masters.
			c.log.WithError(err).Errorf("error creating cluster with VM sizes (master: %s, worker: %s), cannot determine failing profile",
				oc.Properties.MasterProfile.VMSize, oc.Properties.WorkerProfiles[0].VMSize)
			workerIdx++
			if workerIdx >= len(c.Config.CandidateWorkerVMSizes) {
				workerIdx = 0
				masterIdx++
				if masterIdx >= len(c.Config.CandidateMasterVMSizes) {
					return fmt.Errorf("exhausted all VM size combinations: %w", err)
				}
			}
		}
	}
	return nil
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

// getPlatformWIRoleSetsInCosmosDB queries the local RP admin endpoint for
// PlatformWorkloadIdentityRoleSet documents and returns them.
func getPlatformWIRoleSetsInCosmosDB(ctx context.Context) ([]*api.PlatformWorkloadIdentityRoleSet, error) {
	type getRoleSetResponse struct {
		Value []*api.PlatformWorkloadIdentityRoleSet `json:"value"`
	}

	getRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, localDefaultURL+"/admin/platformworkloadidentityrolesets", &bytes.Buffer{})
	if err != nil {
		return nil, fmt.Errorf("error creating get platform WI rolesets request: %w", err)
	}

	getRequest.Header.Set("Content-Type", "application/json")

	getResponse, err := insecureLocalClient().Do(getRequest)
	if err != nil {
		return nil, fmt.Errorf("error couldn't retrieve platform WI role sets in cosmos db: %w", err)
	}

	parsedResponse := getRoleSetResponse{}
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

// ensureDefaultRoleSetInCosmosdb puts a default PlatformWorkloadIdentityRoleSet
// into the cosmos DB via the local RP admin endpoint IF it doesn't already
// contain an entry for the default OpenShift version. It mirrors the behaviour
// of ensureDefaultVersionInCosmosdb but targets the platformworkloadidentityrolesets
// admin endpoint.
func (c *Cluster) ensureDefaultRoleSetInCosmosdb(ctx context.Context) error {
	defaultVersion := version.DefaultInstallStream

	existingRoleSets, err := getPlatformWIRoleSetsInCosmosDB(ctx)
	if err != nil {
		c.log.Warnf("ensureDefaultRoleSetInCosmosdb: getPlatformWIRoleSetsInCosmosDB returned error: %v; will attempt to PUT default", err)
	} else {
		c.log.Infof("ensureDefaultRoleSetInCosmosdb: got %d existing platform WI role sets from local RP", len(existingRoleSets))
		for i, rs := range existingRoleSets {
			var ver string
			if rs != nil {
				ver = rs.Properties.OpenShiftVersion
			}
			c.log.Debugf("ensureDefaultRoleSetInCosmosdb: existingRoleSets[%d].OpenShiftVersion=%s", i, ver)
			if ver == defaultVersion.Version.MinorVersion() {
				c.log.Infof("ensureDefaultRoleSetInCosmosdb: PlatformWorkloadIdentityRoleSet for version %s already in DB; skipping PUT", defaultVersion.Version.MinorVersion())
				return nil
			}
		}
	}

	c.log.Infof("building default payload for OpenShift version %s", defaultVersion.Version.MinorVersion())

	var roleSets []api.PlatformWorkloadIdentityRoleSetProperties
	if err := json.Unmarshal([]byte(c.Config.WorkloadIdentityRoles), &roleSets); err != nil {
		return fmt.Errorf("failed to unmarshal platform workload identity role sets from config: %w", err)
	}

	var defaultRoleSetProperties *api.PlatformWorkloadIdentityRoleSetProperties
	for i := range roleSets {
		if roleSets[i].OpenShiftVersion == defaultVersion.Version.MinorVersion() {
			defaultRoleSetProperties = &roleSets[i]
			break
		}
	}
	if defaultRoleSetProperties == nil {
		return fmt.Errorf("no platform workload identity role set for version %s found", defaultVersion.Version.MinorVersion())
	}

	defaultRoleSet := api.PlatformWorkloadIdentityRoleSet{
		Properties: *defaultRoleSetProperties,
	}

	b, err := json.Marshal(&defaultRoleSet)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, localDefaultURL+"/admin/platformworkloadidentityrolesets", bytes.NewReader(b))
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

// deleteCluster deletes an ARO cluster with retries for transient errors.
// It uses separate timeouts for the overall operation (45m) and each individual
// DeleteAndWait call (35m) to prevent a single slow attempt from exhausting the retry budget.
func (c *Cluster) deleteCluster(ctx context.Context, resourceGroup, clusterName string) error {
	c.log.Printf("deleting cluster %s", clusterName)

	overallTimeout := 45 * time.Minute
	perOperationTimeout := 35 * time.Minute

	overallTimeoutCause := fmt.Errorf("cluster deletion timed out after %s for %s", overallTimeout, clusterName)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, overallTimeout, overallTimeoutCause)
	defer cancel()

	var lastErr error
	// Backoff waits: 0s + 30s + 60s = 90s total. Given perOperationTimeout (35m) and
	// overallTimeout (45m), realistically only ~1 full retry is expected.
	backoff := wait.Backoff{Steps: 3, Duration: 30 * time.Second, Factor: 2.0, Cap: 1 * time.Minute}
	err := wait.ExponentialBackoffWithContext(timeoutCtx, backoff, func() (bool, error) {
		opTimeoutCause := fmt.Errorf("DeleteAndWait timed out after %s for %s", perOperationTimeout, clusterName)
		opCtx, opCancel := context.WithTimeoutCause(timeoutCtx, perOperationTimeout, opTimeoutCause)
		defer opCancel()

		err := c.openshiftclusters.DeleteAndWait(opCtx, resourceGroup, clusterName)
		if err == nil {
			return true, nil
		}

		if timeoutCtx.Err() != nil {
			return false, context.Cause(timeoutCtx)
		}

		if opCtx.Err() != nil {
			c.log.Warnf("operation timed out for cluster %s: %v", clusterName, context.Cause(opCtx))
			lastErr = context.Cause(opCtx)
			return false, nil
		}

		if azureerrors.IsRetryableError(err) {
			c.log.Warnf("retryable error deleting cluster %s, will retry: %v", clusterName, err)
			lastErr = err
			return false, nil
		}
		return false, err
	})
	if err != nil {
		if err == wait.ErrWaitTimeout && lastErr != nil {
			return fmt.Errorf("error deleting cluster %s: %w", clusterName, lastErr)
		}
		return fmt.Errorf("error deleting cluster %s: %w", clusterName, err)
	}
	return nil
}

// checkResourceGroupDeleted polls until the resource group no longer exists or times out.
func (c *Cluster) checkResourceGroupDeleted(ctx context.Context, resourceGroupName string) error {
	c.log.Printf("checking that resource group %s has been deleted", resourceGroupName)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var lastErr error
	err := wait.PollImmediateUntil(5*time.Second, func() (bool, error) {
		_, err := c.groups.Get(timeoutCtx, resourceGroupName)
		if azureerrors.ResourceGroupNotFound(err) {
			c.log.Infof("The resource group %s has been deleted.", resourceGroupName)
			return true, nil
		}
		if err != nil {
			if !azureerrors.IsRetryableError(err) {
				return false, fmt.Errorf("non-retryable error checking resource group %s: %w", resourceGroupName, err)
			}
			lastErr = err
			c.log.Warnf("retryable error checking resource group %s, will retry: %v", resourceGroupName, err)
		} else {
			c.log.Infof("resource group %s still exists, checking for deletion", resourceGroupName)
		}
		return false, nil
	}, timeoutCtx.Done())
	if err != nil {
		if err == wait.ErrWaitTimeout && lastErr != nil {
			return fmt.Errorf("timed out checking for resource group %s to be deleted, last error: %w", resourceGroupName, lastErr)
		}
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("timed out checking for resource group %s to be deleted", resourceGroupName)
		}
		return err
	}
	return nil
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
		AllowVirtualNetworkAccess: pointerutils.ToPtr(true),
		AllowForwardedTraffic:     pointerutils.ToPtr(true),
		UseRemoteGateways:         pointerutils.ToPtr(true),
	}
	rpProp := &sdknetwork.VirtualNetworkPeeringPropertiesFormat{
		RemoteVirtualNetwork: &sdknetwork.SubResource{
			ID: &cluster,
		},
		AllowVirtualNetworkAccess: pointerutils.ToPtr(true),
		AllowForwardedTraffic:     pointerutils.ToPtr(true),
		AllowGatewayTransit:       pointerutils.ToPtr(true),
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
