package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
)

func (i *Installer) updateAzureCloudProvider(ctx context.Context) error {
	spp := i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		acp, err := i.kubernetescli.CoreV1().Secrets("kube-system").Get("azure-cloud-provider", metav1.GetOptions{})
		if err != nil {
			return err
		}
		var config map[string]interface{}
		err = yaml.Unmarshal([]byte(acp.Data["cloud-config"]), &config)
		if err != nil {
			return err
		}

		clientID, _ := config["aadClientId"].(string)
		clientSecret, _ := config["aadClientSecret"].(string)

		if clientID == spp.ClientID && clientSecret == string(spp.ClientSecret) {
			return nil
		}
		config["aadClientId"] = spp.ClientID
		config["aadClientSecret"] = spp.ClientSecret
		acp.Data["cloud-config"], err = yaml.Marshal(config)
		if err != nil {
			return err
		}
		i.log.Info("updated azure-cloud-credentials secret with new values")

		_, err = i.kubernetescli.CoreV1().Secrets("kube-system").Update(acp)
		return err
	})
}

func (i *Installer) updateAzureCredentials(ctx context.Context) error {
	spp := i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := i.kubernetescli.CoreV1().Secrets("kube-system").Get("azure-credentials", metav1.GetOptions{})
		if err != nil {
			return err
		}

		clientID := string(secret.Data["azure_client_id"])
		clientSecret := string(secret.Data["azure_client_secret"])
		if clientID == spp.ClientID && clientSecret == string(spp.ClientSecret) {
			return nil
		}
		secret.Data["aadClientId"] = []byte(spp.ClientID)
		secret.Data["aadClientSecret"] = []byte(spp.ClientSecret)
		i.log.Info("updated azure-credentials secret with new values")

		_, err = i.kubernetescli.CoreV1().Secrets("kube-system").Update(secret)
		return err
	})
}

func (i *Installer) updateRoleAssignments(ctx context.Context) error {
	clusterSPObjectID, err := i.clusterSPObjectID(ctx)
	if err != nil {
		return nil
	}

	roleassignments := authorization.NewRoleAssignmentsClient(i.env.SubscriptionID(), i.fpAuthorizer)
	_, err = roleassignments.Create(ctx, "/subscriptions/"+i.env.SubscriptionID()+"/resourceGroups/"+i.env.ResourceGroup(), uuid.NewV4().String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + i.env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"), // Contributor
			PrincipalID:      to.StringPtr(clusterSPObjectID),
			PrincipalType:    mgmtauthorization.ServicePrincipal,
		},
	})

	if os.Getenv("RP_MODE") == "" {
		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources: []*arm.Resource{
				{
					Resource: &mgmtauthorization.DenyAssignment{
						Name: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
						Type: to.StringPtr("Microsoft.Authorization/denyAssignments"),
						DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
							DenyAssignmentName: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
							Permissions: &[]mgmtauthorization.DenyAssignmentPermission{
								{
									Actions: &[]string{
										"*/action",
										"*/delete",
										"*/write",
									},
									NotActions: &[]string{
										"Microsoft.Network/networkSecurityGroups/join/action",
									},
								},
							},
							Scope: &i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
							Principals: &[]mgmtauthorization.Principal{
								{
									ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
									Type: to.StringPtr("SystemDefined"),
								},
							},
							ExcludePrincipals: &[]mgmtauthorization.Principal{
								{
									ID:   &clusterSPObjectID,
									Type: to.StringPtr("ServicePrincipal"),
								},
							},
							IsSystemProtected: to.BoolPtr(true),
						},
					},
					APIVersion: azureclient.APIVersions["Microsoft.Authorization/denyAssignments"],
				},
			},
		}
		resourceGroupName := fmt.Sprintf("aro-%s", i.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		err = i.deployARMTemplate(ctx, resourceGroupName, "storage", t, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
