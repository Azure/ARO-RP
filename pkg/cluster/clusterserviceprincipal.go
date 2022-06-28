package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/ghodss/yaml"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) createOrUpdateClusterServicePrincipalRBAC(ctx context.Context) error {
	resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
	resourceGroup := stringutils.LastTokenByte(resourceGroupID, '/')
	clusterSPObjectID := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID

	roleAssignments, err := m.roleAssignments.ListForResourceGroup(ctx, resourceGroup, "")
	if err != nil {
		return err
	}

	// We are interested in Resource group scope only (inherited are returned too).
	var toDelete []mgmtauthorization.RoleAssignment
	var found bool
	for _, assignment := range roleAssignments {
		if !strings.EqualFold(*assignment.Scope, resourceGroupID) ||
			strings.HasSuffix(strings.ToLower(*assignment.RoleDefinitionID), strings.ToLower(rbac.RoleOwner)) /* should only matter in development */ {
			continue
		}

		if strings.EqualFold(*assignment.PrincipalID, clusterSPObjectID) &&
			strings.HasSuffix(strings.ToLower(*assignment.RoleDefinitionID), strings.ToLower(rbac.RoleContributor)) {
			found = true
		} else {
			toDelete = append(toDelete, assignment)
		}
	}

	for _, assignment := range toDelete {
		m.log.Infof("deleting role assignment %s", *assignment.Name)
		_, err := m.roleAssignments.Delete(ctx, *assignment.Scope, *assignment.Name)
		if err != nil {
			return err
		}
	}

	err = m.deleteRoleDefinition(ctx)
	if err != nil {
		return err
	}

	if !found {
		m.log.Info("creating cluster service principal role assignment")
		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources:      []*arm.Resource{m.clusterServicePrincipalRBAC()},
		}
		err = arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "storage", t, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) updateAROSecret(ctx context.Context) error {
	var changed bool
	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		//data:
		// cloud-config: <base64 map[string]string with keys 'aadClientId' and 'aadClientSecret'>
		secret, err := m.kubernetescli.CoreV1().Secrets("kube-system").Get(ctx, "azure-cloud-provider", metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) { // we are not in control if secret is not present
				return nil
			}
			return err
		}

		var cf map[string]interface{}
		if secret != nil && secret.Data != nil {
			err = yaml.Unmarshal(secret.Data["cloud-config"], &cf)
			if err != nil {
				return err
			}
			if val, ok := cf["aadClientId"].(string); ok {
				if val != spp.ClientID {
					cf["aadClientId"] = spp.ClientID
					changed = true
				}
			}
			if val, ok := cf["aadClientSecret"].(string); ok {
				if val != string(spp.ClientSecret) {
					cf["aadClientSecret"] = spp.ClientSecret
					changed = true
				}
			}
		}

		if changed {
			data, err := yaml.Marshal(cf)
			if err != nil {
				return err
			}
			secret.Data["cloud-config"] = data

			_, err = m.kubernetescli.CoreV1().Secrets("kube-system").Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// return early if not changed
	if !changed {
		return nil
	}

	// If secret change we need to trigger kube-api-server and kube-controller-manager restarts
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		kAPIServer, err := m.operatorcli.OperatorV1().KubeAPIServers().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}
		kAPIServer.Spec.ForceRedeploymentReason = "Credential rotation " + time.Now().UTC().String()

		_, err = m.operatorcli.OperatorV1().KubeAPIServers().Update(ctx, kAPIServer, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// Log the error and continue.  This code is inherently edge triggered;
		// if we fail and the user retries, we won't re-trigger this code anyway,
		// so it doesn't really help anyone to make this a hard failure
		m.log.Error(err)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		kManager, err := m.operatorcli.OperatorV1().KubeControllerManagers().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}
		kManager.Spec.ForceRedeploymentReason = "Credential rotation " + time.Now().UTC().String()

		_, err = m.operatorcli.OperatorV1().KubeControllerManagers().Update(ctx, kManager, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		m.log.Error(err)
	}
	return nil
}

func (m *manager) updateOpenShiftSecret(ctx context.Context) error {
	var changed bool
	spp := m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		//data:
		// azure_client_id: secret_id
		// azure_client_secret: secret_value
		// azure_tenant_id: tenant_id
		secret, err := m.kubernetescli.CoreV1().Secrets("kube-system").Get(ctx, "azure-credentials", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if string(secret.Data["azure_client_id"]) != spp.ClientID {
			secret.Data["azure_client_id"] = []byte(spp.ClientID)
			changed = true
		}

		if string(secret.Data["azure_client_secret"]) != string(spp.ClientSecret) {
			secret.Data["azure_client_secret"] = []byte(spp.ClientSecret)
			changed = true
		}

		if string(secret.Data["azure_tenant_id"]) != m.subscriptionDoc.Subscription.Properties.TenantID {
			secret.Data["azure_tenant_id"] = []byte(m.subscriptionDoc.Subscription.Properties.TenantID)
			changed = true
		}

		if changed {
			_, err = m.kubernetescli.CoreV1().Secrets("kube-system").Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// return early if not changed
	if !changed {
		return nil
	}

	// restart cloud credentials operator to trigger rotation
	err = m.kubernetescli.CoreV1().Pods("openshift-cloud-credential-operator").DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "app=cloud-credential-operator",
	})
	if err != nil {
		// Log the error and continue.  This code is inherently edge triggered;
		// if we fail and the user retries, we won't re-trigger this code anyway,
		// so it doesn't really help anyone to make this a hard failure
		m.log.Error(err)
	}
	return nil
}
