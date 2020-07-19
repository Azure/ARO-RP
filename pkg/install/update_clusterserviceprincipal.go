package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
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
		err = json.Unmarshal([]byte(acp.Data["cloud-config"]), &config)
		if err != nil {
			return err
		}

		// don't panic if aadClientId or aadClientSecret is not a string
		clientID, _ := config["aadClientId"].(string)
		clientSecret, _ := config["aadClientSecret"].(string)
		if clientID == spp.ClientID && clientSecret == string(spp.ClientSecret) {
			return nil
		}
		config["aadClientId"] = spp.ClientID
		config["aadClientSecret"] = spp.ClientSecret
		acp.Data["cloud-config"], err = json.Marshal(config)
		if err != nil {
			return err
		}

		_, err = i.kubernetescli.CoreV1().Secrets("kube-system").Update(acp)
		return err
	})
}

func (i *Installer) updateOpenShiftCloudProviderConfig(ctx context.Context) error {
	spp := i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cpc, err := i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Get("cloud-provider-config", metav1.GetOptions{})
		if err != nil {
			return err
		}

		var config map[string]interface{}
		err = json.Unmarshal([]byte(cpc.Data["config"]), &config)
		if err != nil {
			return err
		}
		tenantID, ok := config["tenantId"].(string)
		if ok && tenantID == spp.TenantID {
			return nil
		}
		config["tenantID"] = spp.TenantID
		b, err := json.Marshal(config)
		if err != nil {
			return err
		}
		cpc.Data["config"] = string(b)
		_, err = i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Update(cpc)
		return err
	})
}

func (i *Installer) clusterSPObjectID(ctx context.Context) (string, error) {
	var result string

	spp := &i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	token, err := aad.GetToken(ctx, i.log, i.doc.OpenShiftCluster, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return "", err
	}

	spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

	applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	// NOTE: Do not override err with the error returned by wait.PollImmediateUntil.
	// Doing this will not propagate the latest error to the user in case when wait exceeds the timeout
	wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var res azgraphrbac.ServicePrincipalObjectResult
		res, err = applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
		if err != nil {
			if strings.Contains(err.Error(), "Authorization_IdentityNotFound") {
				i.log.Info(err)
				return false, nil
			}

			return false, err
		}

		result = *res.Value
		return true, nil
	}, timeoutCtx.Done())

	return result, err
}

func (i *Installer) updateRoleAssignments(ctx context.Context) error {
	clusterSPObjectID, err := i.clusterSPObjectID(ctx)
	if err != nil {
		return nil
	}
	// TODO shouldn't this be the armAuthoriser?
	roleassignments := authorization.NewRoleAssignmentsClient(i.env.SubscriptionID(), i.fpAuthorizer)

	_, err = roleassignments.Create(ctx, "/subscriptions/"+i.env.SubscriptionID()+"/resourceGroups/"+i.env.ResourceGroup(), uuid.NewV4().String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + i.env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"),
			PrincipalID:      to.StringPtr(clusterSPObjectID),
			PrincipalType:    mgmtauthorization.ServicePrincipal,
		},
	})
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "RoleAssignmentExists" {
			err = nil
		}
	}
	/*	TODO, we also need to do this but there does not seem to be a create/update api
		if _, ok := i.env.(env.Dev); !ok {

			Resource: &mgmtauthorization.DenyAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
				Type: to.StringPtr("Microsoft.Authorization/denyAssignments"),
				DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
					ExcludePrincipals: &[]mgmtauthorization.Principal{
						{
							ID:   &clusterSPObjectID,
							Type: to.StringPtr("ServicePrincipal"),
						},
					},
					IsSystemProtected: to.BoolPtr(true),
				},
			},
		}
	*/
	return nil
}
