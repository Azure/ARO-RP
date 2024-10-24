package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/oidcbuilder"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var nsgNotReadyErrorRegex = regexp.MustCompile("Resource.*networkSecurityGroups.*referenced by resource.*not found")

const storageServiceEndpoint = "Microsoft.Storage"

func (m *manager) createDNS(ctx context.Context) error {
	return m.dns.Create(ctx, m.doc.OpenShiftCluster)
}

func (m *manager) createOIDC(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil
	}

	// OIDC Storage Web Endpoint need to be determined for Development environments
	var oidcEndpoint string
	if m.env.FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint) {
		properties, err := m.rpBlob.GetContainerProperties(ctx, m.env.ResourceGroup(), m.env.OIDCStorageAccountName(), oidcbuilder.WebContainer)
		if err != nil {
			return err
		}
		oidcEndpoint = *properties.Properties.PrimaryEndpoints.Web
	} else {
		// For Production Azure Front Door Endpoint will be the OIDC Endpoint
		oidcEndpoint = m.env.OIDCEndpoint()
	}
	oidcBuilder, err := oidcbuilder.NewOIDCBuilder(m.env, oidcEndpoint, oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, m.doc.ID))
	if err != nil {
		return err
	}

	blobsClient, err := m.rpBlob.GetBlobsClient(oidcBuilder.GetBlobContainerURL())
	if err != nil {
		return err
	}

	err = oidcBuilder.EnsureOIDCDocs(ctx, blobsClient)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer = pointerutils.ToPtr(api.OIDCIssuer(oidcBuilder.GetEndpointUrl()))
		doc.OpenShiftCluster.Properties.ClusterProfile.BoundServiceAccountSigningKey = pointerutils.ToPtr(api.SecureString(oidcBuilder.GetPrivateKey()))
		return nil
	})

	return err
}

func (m *manager) ensureInfraID(ctx context.Context) (err error) {
	if m.doc.OpenShiftCluster.Properties.InfraID != "" {
		return err
	}
	// generate an infra ID that is 27 characters long with 5 bytes of them random
	infraID := generateInfraID(strings.ToLower(m.doc.OpenShiftCluster.Name), 27, 5)
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.InfraID = infraID
		return nil
	})
	return err
}

func (m *manager) ensureResourceGroup(ctx context.Context) (err error) {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	group := mgmtfeatures.ResourceGroup{}

	// Retain the existing resource group configuration (such as tags) if it exists
	group, err = m.resourceGroups.Get(ctx, resourceGroup)
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); !ok || detailedErr.StatusCode != http.StatusNotFound {
			return err
		}

		// set field values if the RG doesn't exist
		group.Location = &m.doc.OpenShiftCluster.Location
		group.ManagedBy = &m.doc.OpenShiftCluster.ID
	}

	resourceGroupAlreadyExistsError := &api.CloudError{
		StatusCode: http.StatusBadRequest,
		CloudErrorBody: &api.CloudErrorBody{
			Code: api.CloudErrorCodeClusterResourceGroupAlreadyExists,
			Message: "Resource group " + m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID +
				" must not already exist.",
		},
	}

	// If managedBy or location don't match, return an error that RG must not already exist
	if group.Location == nil || !strings.EqualFold(*group.Location, m.doc.OpenShiftCluster.Location) {
		return resourceGroupAlreadyExistsError
	}

	if group.ManagedBy == nil || !strings.EqualFold(*group.ManagedBy, m.doc.OpenShiftCluster.ID) {
		return resourceGroupAlreadyExistsError
	}

	// HACK: set purge=true on dev clusters so our purger wipes them out since there is not deny assignment in place
	if m.env.IsLocalDevelopmentMode() {
		if group.Tags == nil {
			group.Tags = map[string]*string{}
		}
		group.Tags["purge"] = to.StringPtr("true")
	}

	// According to https://stackoverflow.microsoft.com/a/245391/62320,
	// re-PUTting our RG should re-create RP RBAC after a customer subscription
	// migrates between tenants.
	_, err = m.resourceGroups.CreateOrUpdate(ctx, resourceGroup, group)

	var serviceError *azure.ServiceError
	// CreateOrUpdate wraps DetailedError wrapping a *RequestError (if error generated in ResourceGroup CreateOrUpdateResponder at least)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok {
			serviceError = requestErr.ServiceError
		}
	}

	// TODO [gv]: Keeping this for retro-compatibility, but probably this can be removed
	if requestErr, ok := err.(*azure.RequestError); ok {
		serviceError = requestErr.ServiceError
	}

	if serviceError != nil && serviceError.Code == "RequestDisallowedByPolicy" {
		// if request was disallowed by policy, inform user so they can take appropriate action
		b, _ := json.Marshal(serviceError)
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: string(b),
					},
				},
			},
		}
	}
	if err != nil {
		return err
	}

	return m.env.EnsureARMResourceGroupRoleAssignment(ctx, resourceGroup)
}

func (m *manager) deployBaseResourceTemplate(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix
	azureRegion := strings.ToLower(m.doc.OpenShiftCluster.Location) // Used in k8s object names, so must pass DNS-1123 validation

	ocpSubnets, err := m.subnetsWithServiceEndpoint(ctx, storageServiceEndpoint)
	if err != nil {
		return err
	}

	resources := []*arm.Resource{
		m.storageAccount(clusterStorageAccountName, azureRegion, ocpSubnets, true, true),
		m.storageAccountBlobContainer(clusterStorageAccountName, graph.IgnitionContainer),
		m.storageAccountBlobContainer(clusterStorageAccountName, graph.GraphContainer),
		m.storageAccount(m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName, azureRegion, ocpSubnets, true, false),
		m.storageAccountBlobContainer(m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName, "image-registry"),
		m.clusterNSG(infraID, azureRegion),
		m.networkPrivateLinkService(azureRegion),
		m.networkInternalLoadBalancer(azureRegion),
	}

	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		r, err := m.platformWorkloadIdentityRBAC()
		if err != nil {
			return err
		}

		resources = append(resources, r...)
	} else {
		resources = append(resources, m.clusterServicePrincipalRBAC())
	}

	// Create a public load balancer routing if needed
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeLoadbalancer {
		m.newPublicLoadBalancer(ctx, &resources)

		// If the cluster is public we still want the default public IP address
		if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
			resources = append(resources,
				m.networkPublicIPAddress(azureRegion, infraID+"-default-v4"),
			)
		}
	}

	if m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		resources = append(resources,
			m.networkPrivateEndpoint(),
		)
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
	}

	if !m.env.FeatureIsSet(env.FeatureDisableDenyAssignments) {
		t.Resources = append(t.Resources, m.denyAssignment())
	}

	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		storageBlobContributorRBAC, err := m.fpspStorageBlobContributorRBAC(clusterStorageAccountName, m.fpServicePrincipalID)
		if err != nil {
			return err
		}
		t.Resources = append(t.Resources, storageBlobContributorRBAC)
	}

	return arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "storage", t, nil)
}

func (m *manager) newPublicLoadBalancer(ctx context.Context, resources *[]*arm.Resource) {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	azureRegion := strings.ToLower(m.doc.OpenShiftCluster.Location) // Used in k8s object names, so must pass DNS-1123 validation

	var outboundIPs []api.ResourceReference
	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		*resources = append(*resources,
			m.networkPublicIPAddress(azureRegion, infraID+"-pip-v4"),
		)
		if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
			outboundIPs = append(outboundIPs, api.ResourceReference{ID: m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/publicIPAddresses/" + infraID + "-pip-v4"})
		}
	}
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		for i := len(outboundIPs); i < m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count; i++ {
			ipName := genManagedOutboundIPName()
			*resources = append(*resources, m.networkPublicIPAddress(azureRegion, ipName))
			outboundIPs = append(outboundIPs, api.ResourceReference{ID: m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/publicIPAddresses/" + ipName})
		}
	}
	m.patchEffectiveOutboundIPs(ctx, outboundIPs)

	*resources = append(*resources,
		m.networkPublicLoadBalancer(azureRegion, outboundIPs),
	)
}

// subnetsWithServiceEndpoint returns a unique slice of subnet resource IDs that have the corresponding
// service endpoint
func (m *manager) subnetsWithServiceEndpoint(ctx context.Context, serviceEndpoint string) ([]string, error) {
	subnetsMap := map[string]struct{}{}

	subnetsMap[m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID] = struct{}{}
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)
	for _, v := range workerProfiles {
		// don't fail empty worker profiles/subnet IDs as they're not valid
		if v.SubnetID == "" {
			continue
		}

		subnetsMap[strings.ToLower(v.SubnetID)] = struct{}{}
	}

	subnets := []string{}
	for subnetId := range subnetsMap {
		// We purposefully fail if we can't fetch the subnet as the FPSP most likely
		// lost read permission over the subnet.
		subnet, err := m.subnet.Get(ctx, subnetId)
		if err != nil {
			return nil, err
		}

		if subnet.SubnetPropertiesFormat == nil || subnet.ServiceEndpoints == nil {
			continue
		}

		for _, endpoint := range *subnet.ServiceEndpoints {
			if endpoint.Service != nil && strings.EqualFold(*endpoint.Service, serviceEndpoint) && endpoint.Locations != nil {
				for _, loc := range *endpoint.Locations {
					if loc == "*" || strings.EqualFold(loc, m.doc.OpenShiftCluster.Location) {
						subnets = append(subnets, subnetId)
					}
				}
			}
		}
	}

	return subnets, nil
}

// attachNSGs attaches NSGs to the cluster subnets, if preconfigured NSG is not
// enabled. This method is suitable for use with steps, and has default
// timeout/polls set.
func (m *manager) attachNSGs(ctx context.Context) error {
	// Since we need to guard against the case where NSGs are not ready
	// immediately after creation, we can have a relatively short retry period
	// of 30s and timeout of 3m. These numbers were chosen via a
	// highly-non-specific and data-adjacent process (picking them because they
	// seemed decent enough).
	//
	// If we get the NSG not-ready error after 3 minutes, it's unusual enough
	// that we should be raising it as an issue rather than tolerating it.
	return m._attachNSGs(ctx, 3*time.Minute, 30*time.Second)
}

// _attachNSGs attaches NSGs to the cluster subnets, if preconfigured NSG is not
// enabled. timeout and pollInterval are provided as arguments for testing
// reasons.
func (m *manager) _attachNSGs(ctx context.Context, timeout time.Duration, pollInterval time.Duration) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGEnabled {
		return nil
	}
	var innerErr error

	workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)
	workerSubnetId := workerProfiles[0].SubnetID

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// This polling function protects the case below where the NSG might not be
	// ready to be referenced. We don't guard against trying to re-attach the
	// NSG since the inner loop is tolerant of that, and since we are attaching
	// the same NSG the only allowed failure case is when the NSG cannot be
	// attached to begin with, so it shouldn't happen in practice.
	_ = wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		var c bool
		c, innerErr = func() (bool, error) {
			for _, subnetID := range []string{
				m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
				workerSubnetId,
			} {
				m.log.Printf("attaching network security group to subnet %s", subnetID)
				// TODO: there is probably an undesirable race condition here - check if etags can help.

				// We use the outer context, not the timeout context, as we do not want
				// to time out the condition function itself, only stop retrying once
				// timeoutCtx's timeout has fired.
				s, err := m.subnet.Get(ctx, subnetID)
				if err != nil {
					return false, err
				}

				if s.SubnetPropertiesFormat == nil {
					s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
				}

				nsgID, err := apisubnet.NetworkSecurityGroupID(m.doc.OpenShiftCluster, subnetID)
				if err != nil {
					return false, err
				}

				// Sometimes we get into the race condition between external services modifying
				// subnets and our validation code. We try to catch this early, but
				// these errors is propagated to make the user-facing error more clear incase
				// modification happened after we ran validation code and we lost the race
				if s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
					if strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
						continue
					}

					return false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must not have a network security group attached.", subnetID)
				}

				s.SubnetPropertiesFormat.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr(nsgID),
				}

				// Because we attempt to attach the NSG immediately after the base resource deployment
				// finishes, the NSG is sometimes not yet ready to be referenced and used, causing
				// an error to occur here. So if this particular error occurs, return nil to retry.
				// But if some other type of error occurs, just return that error.
				err = m.subnet.CreateOrUpdate(ctx, subnetID, s)
				if err != nil {
					if nsgNotReadyErrorRegex.MatchString(err.Error()) {
						return false, nil
					}
					return false, err
				}
			}
			return true, nil
		}()

		return c, innerErr
	}, timeoutCtx.Done())

	return innerErr
}

func (m *manager) setMasterSubnetPolicies(ctx context.Context) error {
	// TODO: there is probably an undesirable race condition here - check if etags can help.
	subnetId := m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID
	s, err := m.subnet.Get(ctx, subnetId)
	if err != nil {
		return err
	}

	if s.SubnetPropertiesFormat == nil {
		s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
	}

	if m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		s.SubnetPropertiesFormat.PrivateEndpointNetworkPolicies = to.StringPtr("Disabled")
	}
	s.SubnetPropertiesFormat.PrivateLinkServiceNetworkPolicies = to.StringPtr("Disabled")

	err = m.subnet.CreateOrUpdate(ctx, subnetId, s)

	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if strings.Contains(detailedErr.Original.Error(), "RequestDisallowedByPolicy") {
			return &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code: api.CloudErrorCodeRequestDisallowedByPolicy,
					Message: fmt.Sprintf("Resource %s was disallowed by policy.",
						subnetId[strings.LastIndex(subnetId, "/")+1:],
					),
					Details: []api.CloudErrorBody{
						{
							Code: api.CloudErrorCodeRequestDisallowedByPolicy,
							Message: fmt.Sprintf("Policy definition : %s\nPolicy Assignment : %s",
								regexp.MustCompile(`policyDefinitionName":"([^"]+)"`).FindStringSubmatch(detailedErr.Original.Error())[1],
								regexp.MustCompile(`policyAssignmentName":"([^"]+)"`).FindStringSubmatch(detailedErr.Original.Error())[1],
							),
						},
					},
				},
			}
		}
	}
	return err
}

func (m *manager) federateIdentityCredentials(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil
	}

	if m.doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer == nil {
		return errors.New("OIDCIssuer is nil")
	}

	issuer := to.StringPtr((string)(*m.doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer))

	platformWIRolesByRoleName := m.platformWorkloadIdentityRolesByVersion.GetPlatformWorkloadIdentityRolesByRoleName()
	platformWorkloadIdentities := m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities

	for name, identity := range platformWorkloadIdentities {
		identityResourceId, err := azure.ParseResourceID(identity.ResourceID)
		if err != nil {
			return err
		}

		platformWIRole, exists := platformWIRolesByRoleName[name]
		if !exists {
			continue
		}

		for _, sa := range platformWIRole.ServiceAccounts {
			federatedIdentityCredentialResourceName, err := m.getPlatformWorkloadIdentityFederatedCredName(sa, identity)
			if err != nil {
				return err
			}

			_, err = m.clusterMsiFederatedIdentityCredentials.CreateOrUpdate(
				ctx,
				identityResourceId.ResourceGroup,
				identityResourceId.ResourceName,
				federatedIdentityCredentialResourceName,
				armmsi.FederatedIdentityCredential{
					Properties: &armmsi.FederatedIdentityCredentialProperties{
						Audiences: []*string{to.StringPtr("openshift")},
						Issuer:    issuer,
						Subject:   to.StringPtr(sa),
					},
				},
				&armmsi.FederatedIdentityCredentialsClientCreateOrUpdateOptions{},
			)

			if err != nil {
				return err
			}
		}
	}
	return nil
}

// generateInfraID take base and returns a ID that
// - is of length maxLen
// - contains randomLen random bytes
// - only contains `alphanum` or `-`
// see openshift/installer/pkg/asset/installconfig/clusterid.go for original implementation
func generateInfraID(base string, maxLen int, randomLen int) string {
	maxBaseLen := maxLen - (randomLen + 1)

	// replace all characters that are not `alphanum` or `-` with `-`
	re := regexp.MustCompile("[^A-Za-z0-9-]")
	base = re.ReplaceAllString(base, "-")

	// replace all multiple dashes in a sequence with single one.
	re = regexp.MustCompile(`-{2,}`)
	base = re.ReplaceAllString(base, "-")

	// truncate to maxBaseLen
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	base = strings.TrimRight(base, "-")

	// add random chars to the end to randomize
	return fmt.Sprintf("%s-%s", base, utilrand.String(randomLen))
}
