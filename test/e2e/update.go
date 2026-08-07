package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	armredhatopenshift "github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

var _ = Describe("Update clusters", func() {
	It("must replace the ARO operator's CredentialsRequest if it has been deleted", func(ctx context.Context) {
		crNamespacedName := types.NamespacedName{
			Namespace: "openshift-cloud-credential-operator",
			Name:      "openshift-azure-operator",
		}

		By("deleting the CredentialsRequest")
		cr := &cloudcredentialv1.CredentialsRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "openshift-cloud-credential-operator",
				Name:      "openshift-azure-operator",
			},
		}
		err := clients.Client.Delete(ctx, cr)
		Expect(err).NotTo(HaveOccurred())

		By("sending the PATCH request to update the cluster")
		err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, armredhatopenshift.OpenShiftClusterUpdate{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that the CredentialsRequest has been recreated")
		cr = &cloudcredentialv1.CredentialsRequest{}
		err = clients.Client.Get(ctx, crNamespacedName, cr)
		Expect(err).NotTo(HaveOccurred())
	})

	It("must restart the aro-operator-master Deployment", func(ctx context.Context) {
		if isMiwi {
			Skip("This test is not relevant for miwi clusters")
		}
		By("saving the current revision of the aro-operator-master Deployment")
		getFunc := clients.Kubernetes.AppsV1().Deployments("openshift-azure-operator").Get
		deployment := GetK8sObjectWithRetry(ctx, getFunc, "aro-operator-master", metav1.GetOptions{})

		Expect(deployment.ObjectMeta.Annotations).To(HaveKey("deployment.kubernetes.io/revision"))

		oldRevision, err := strconv.Atoi(deployment.Annotations["deployment.kubernetes.io/revision"])
		Expect(err).NotTo(HaveOccurred())

		By("sending the PATCH request to update the cluster")
		err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, armredhatopenshift.OpenShiftClusterUpdate{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that the aro-operator-master Deployment was restarted")
		deployment = GetK8sObjectWithRetry(ctx, getFunc, "aro-operator-master", metav1.GetOptions{})

		Expect(deployment.Spec.Template.Annotations).To(HaveKey("kubectl.kubernetes.io/restartedAt"))
		Expect(deployment.ObjectMeta.Annotations).To(HaveKey("deployment.kubernetes.io/revision"))

		newRevision, err := strconv.Atoi(deployment.Annotations["deployment.kubernetes.io/revision"])
		Expect(err).NotTo(HaveOccurred())
		Expect(newRevision).To(Equal(oldRevision + 1))
	})

	It("should successfully replace platform workload identity", func(ctx context.Context) {
		if !isMiwi {
			Skip("This test is only relevant for workload identity clusters")
		}

		federatedCredentialRoleDefinitionID := "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleAzureRedHatOpenShiftFederatedCredentialRole

		By("getting the current cluster to read existing platform workload identities")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(oc.Properties.PlatformWorkloadIdentityProfile).NotTo(BeNil())
		Expect(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities).NotTo(BeEmpty())

		By("picking an operator identity to replace")
		var operatorName string
		for name := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			operatorName = name
			break
		}
		replacementIdentityName := operatorName + "-e2e-replace"
		By(fmt.Sprintf("targeting operator %q for identity replacement", operatorName))

		By("looking up the operator's role definition from platform workload identity role sets")
		clusterVersion := *oc.Properties.ClusterProfile.Version
		lastDot := strings.LastIndex(clusterVersion, ".")
		Expect(lastDot).To(BeNumerically(">", 0), "cluster version %q is not in x.y.z format", clusterVersion)
		clusterMinorVersion := clusterVersion[:lastDot]

		var operatorRoleDefinitionID string
		roleSetsPager := clients.PlatformWorkloadIdentityRoleSets.NewListPager(*oc.Location, nil)
		for roleSetsPager.More() {
			roleSetsPage, err := roleSetsPager.NextPage(ctx)
			Expect(err).NotTo(HaveOccurred())
			for _, roleSet := range roleSetsPage.Value {
				if roleSet.Properties != nil && roleSet.Properties.OpenShiftVersion != nil && *roleSet.Properties.OpenShiftVersion == clusterMinorVersion {
					for _, role := range roleSet.Properties.PlatformWorkloadIdentityRoles {
						if role.OperatorName != nil && *role.OperatorName == operatorName {
							operatorRoleDefinitionID = *role.RoleDefinitionID
							break
						}
					}
				}
				if operatorRoleDefinitionID != "" {
					break
				}
			}
			if operatorRoleDefinitionID != "" {
				break
			}
		}
		Expect(operatorRoleDefinitionID).NotTo(BeEmpty(), "could not find role definition for operator %s", operatorName)

		By("reading the cluster identity principal ID for federated credential role assignment")
		Expect(oc.Identity).NotTo(BeNil())
		var clusterIdentityPrincipalID string
		for _, identity := range oc.Identity.UserAssignedIdentities {
			clusterIdentityPrincipalID = *identity.PrincipalID
			break
		}
		Expect(clusterIdentityPrincipalID).NotTo(BeEmpty())

		By("checking the operator's role definition for additional scope requirements")
		roleDef, err := clients.RoleDefinitions.GetByID(ctx, operatorRoleDefinitionID)
		Expect(err).NotTo(HaveOccurred())
		var requiresDESPermission, requiresRouteTablePermission bool
		if roleDef.RoleDefinitionProperties != nil && roleDef.Permissions != nil {
			for _, perm := range *roleDef.Permissions {
				if perm.Actions == nil {
					continue
				}
				for _, action := range *perm.Actions {
					if strings.Contains(action, "Microsoft.Compute/diskEncryptionSets") {
						requiresDESPermission = true
					}
					if strings.Contains(action, "Microsoft.Network/routeTables") {
						requiresRouteTablePermission = true
					}
				}
			}
		}

		By("deriving the VNet scope from the master subnet")
		masterSubnetID := *oc.Properties.MasterProfile.SubnetID
		subnetsIdx := strings.LastIndex(masterSubnetID, "/subnets/")
		Expect(subnetsIdx).To(BeNumerically(">", 0), "master subnet ID %q does not contain /subnets/ segment", masterSubnetID)
		vnetScope := masterSubnetID[:subnetsIdx]

		By("creating a replacement managed identity")
		msiResp, err := clients.UserAssignedIdentities.CreateOrUpdate(ctx, vnetResourceGroup, replacementIdentityName, armmsi.Identity{
			Location: oc.Location,
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(msiResp.ID).NotTo(BeNil())
		Expect(msiResp.Properties).NotTo(BeNil())
		Expect(msiResp.Properties.PrincipalID).NotTo(BeNil())
		replacementResourceID := *msiResp.ID
		replacementPrincipalID := *msiResp.Properties.PrincipalID

		DeferCleanup(func(ctx context.Context) {
			By("cleaning up the original identity and its role assignments")
			originalMsi, err := clients.UserAssignedIdentities.Get(ctx, vnetResourceGroup, operatorName, nil)
			if err != nil {
				return
			}
			if originalMsi.Properties != nil && originalMsi.Properties.PrincipalID != nil {
				roleAssignments, err := clients.RoleAssignments.ListForResourceGroup(ctx, vnetResourceGroup, fmt.Sprintf("principalId eq '%s'", *originalMsi.Properties.PrincipalID))
				if err == nil {
					for _, ra := range roleAssignments {
						if strings.HasPrefix(strings.ToLower(*ra.Scope), strings.ToLower("/subscriptions/"+_env.SubscriptionID()+"/resourceGroups/"+vnetResourceGroup)) {
							_, _ = clients.RoleAssignments.Delete(ctx, *ra.Scope, *ra.Name)
						}
					}
				}
			}
			_, _ = clients.UserAssignedIdentities.Delete(ctx, vnetResourceGroup, operatorName, nil)
		})

		By("assigning the operator's role to the replacement identity at VNet scope")
		_, err = clients.RoleAssignments.Create(ctx, vnetScope, uuid.DefaultGenerator.Generate(), mgmtauthorization.RoleAssignmentCreateParameters{
			RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
				RoleDefinitionID: &operatorRoleDefinitionID,
				PrincipalID:      &replacementPrincipalID,
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		if requiresDESPermission && oc.Properties.MasterProfile.DiskEncryptionSetID != nil && *oc.Properties.MasterProfile.DiskEncryptionSetID != "" {
			By("assigning the operator's role to the replacement identity at DiskEncryptionSet scope")
			_, err = clients.RoleAssignments.Create(ctx, *oc.Properties.MasterProfile.DiskEncryptionSetID, uuid.DefaultGenerator.Generate(), mgmtauthorization.RoleAssignmentCreateParameters{
				RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
					RoleDefinitionID: &operatorRoleDefinitionID,
					PrincipalID:      &replacementPrincipalID,
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		}

		if requiresRouteTablePermission {
			By("looking up route tables from the cluster subnets")

			vnetName := stringutils.LastTokenByte(vnetScope, '/')
			subnetIDs := []string{*oc.Properties.MasterProfile.SubnetID}
			if oc.Properties.WorkerProfiles != nil {
				for _, wp := range oc.Properties.WorkerProfiles {
					subnetIDs = append(subnetIDs, *wp.SubnetID)
				}
			}

			routeTableIDs := map[string]struct{}{}
			for _, subnetID := range subnetIDs {
				subnetName := stringutils.LastTokenByte(subnetID, '/')
				subnetResp, err := clients.Subnet.Get(ctx, vnetResourceGroup, vnetName, subnetName, nil)
				Expect(err).NotTo(HaveOccurred())
				if subnetResp.Properties != nil && subnetResp.Properties.RouteTable != nil && subnetResp.Properties.RouteTable.ID != nil {
					routeTableIDs[*subnetResp.Properties.RouteTable.ID] = struct{}{}
				}
			}
			Expect(routeTableIDs).NotTo(BeEmpty(), "requiresRouteTablePermission=true but no route table was found on cluster subnets")

			for routeTableID := range routeTableIDs {
				By("assigning the operator's role to the replacement identity at route table scope")
				_, err = clients.RoleAssignments.Create(ctx, routeTableID, uuid.DefaultGenerator.Generate(), mgmtauthorization.RoleAssignmentCreateParameters{
					RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
						RoleDefinitionID: &operatorRoleDefinitionID,
						PrincipalID:      &replacementPrincipalID,
						PrincipalType:    mgmtauthorization.ServicePrincipal,
					},
				})
				Expect(err).NotTo(HaveOccurred())
			}
		}

		By("assigning the federated credential role to the cluster identity at the scope of the replacement identity")
		_, err = clients.RoleAssignments.Create(ctx, replacementResourceID, uuid.DefaultGenerator.Generate(), mgmtauthorization.RoleAssignmentCreateParameters{
			RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
				RoleDefinitionID: &federatedCredentialRoleDefinitionID,
				PrincipalID:      &clusterIdentityPrincipalID,
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		By("sending the PATCH request to replace the operator identity")
		err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, armredhatopenshift.OpenShiftClusterUpdate{
			Properties: &armredhatopenshift.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &armredhatopenshift.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]*armredhatopenshift.PlatformWorkloadIdentity{
						operatorName: {
							ResourceID: &replacementResourceID,
						},
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		By("verifying the identity was replaced")
		oc, err = clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(*oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[operatorName].ResourceID).To(Equal(replacementResourceID))
	})

	// This tests the API which is most commonly generated by
	// az resource tag --tags key=value --ids /subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.RedHatOpenShift/openShiftClusters/xxx
	It("must be possible to set tags on a cluster resource via PUT", func(ctx context.Context) {
		value := strconv.Itoa(rand.Int())

		By("getting cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(oc.Tags).NotTo(HaveKeyWithValue("key", &value))

		By("adding a new test tag")
		if oc.Tags == nil {
			oc.Tags = map[string]*string{}
		}
		oc.Tags["key"] = &value

		By("sending the PUT request to update the resource")
		err = clients.OpenshiftClusters.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, oc.OpenShiftCluster)
		Expect(err).NotTo(HaveOccurred())

		By("getting the cluster resource")
		oc, err = clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())

		By("checking that the tag has expected value")
		Expect(oc.Tags).To(HaveKeyWithValue("key", &value))
	})
})

var _ = Describe("Update cluster Managed Outbound IPs", func() {
	var lbName string
	var rgName string

	_ = BeforeEach(func(ctx context.Context) {
		By("ensuring the public loadbalancer starts with one outbound IP")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())

		lbName, err = getInfraID(ctx)
		Expect(err).NotTo(HaveOccurred())

		rgName = stringutils.LastTokenByte(*oc.Properties.ClusterProfile.ResourceGroupID, '/')
		resp, err := clients.LoadBalancers.Get(ctx, rgName, lbName, nil)
		Expect(err).NotTo(HaveOccurred())

		if getOutboundIPsCount(resp.LoadBalancer) != 1 {
			By("sending the PATCH request to set ManagedOutboundIPs.Count to 1")
			err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, newManagedOutboundIPUpdateBody(1))
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("must be possible to increase and decrease IP Addresses on the public loadbalancer", func(ctx context.Context) {
		By("sending the PATCH request to increase Managed Outbound IPs")
		err := clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, newManagedOutboundIPUpdateBody(5))
		Expect(err).NotTo(HaveOccurred())

		By("getting the cluster resource")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())

		By("checking effectiveOutboundIPs has been updated")
		Expect(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs).To(HaveLen(5))

		By("checking outbound-rule-4 has required number IPs")
		resp, err := clients.LoadBalancers.Get(ctx, rgName, lbName, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(getOutboundIPsCount(resp.LoadBalancer)).To(Equal(5))

		By("sending the PUT request to decrease Managed Outbound IPs")
		oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count = pointerutils.ToPtr(int32(1))
		err = clients.OpenshiftClusters.CreateOrUpdateAndWait(ctx, vnetResourceGroup, clusterName, oc.OpenShiftCluster)
		Expect(err).NotTo(HaveOccurred())

		By("getting the cluster resource")
		oc, err = clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName, nil)
		Expect(err).NotTo(HaveOccurred())

		By("checking effectiveOutboundIPs has been updated")
		Expect(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs).To(HaveLen(1))

		By("checking outbound-rule-4 has required number of IPs")
		resp, err = clients.LoadBalancers.Get(ctx, rgName, lbName, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(getOutboundIPsCount(resp.LoadBalancer)).To(Equal(1))
	})
})

func getInfraID(ctx context.Context) (string, error) {
	co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return co.Spec.InfraID, err
}

func newManagedOutboundIPUpdateBody(managedOutboundIPCount int32) armredhatopenshift.OpenShiftClusterUpdate {
	return armredhatopenshift.OpenShiftClusterUpdate{
		Properties: &armredhatopenshift.OpenShiftClusterProperties{
			NetworkProfile: &armredhatopenshift.NetworkProfile{
				LoadBalancerProfile: &armredhatopenshift.LoadBalancerProfile{
					ManagedOutboundIPs: &armredhatopenshift.ManagedOutboundIPs{
						Count: pointerutils.ToPtr(managedOutboundIPCount),
					},
				},
			},
		},
	}
}

func getOutboundIPsCount(lb armnetwork.LoadBalancer) int {
	numOfIPs := 0
	for _, obRule := range lb.Properties.OutboundRules {
		if *obRule.Name == "outbound-rule-v4" {
			numOfIPs = len(obRule.Properties.FrontendIPConfigurations)
		}
	}
	return numOfIPs
}
