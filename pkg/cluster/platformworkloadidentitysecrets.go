package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const azureFederatedTokenFileLocation = "/var/run/secrets/openshift/serviceaccount/token"

func (m *manager) generatePlatformWorkloadIdentitySecrets() ([]corev1.Secret, error) {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil, fmt.Errorf("generatePlatformWorkloadIdentitySecrets called for a CSP cluster")
	}

	subscriptionId := m.subscriptionDoc.ID
	tenantId := m.subscriptionDoc.Subscription.Properties.TenantID
	region := m.doc.OpenShiftCluster.Location

	roles := m.platformWorkloadIdentityRolesByVersion.GetPlatformWorkloadIdentityRolesByRoleName()

	secrets := []corev1.Secret{}
	for _, identity := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		if role, ok := roles[identity.OperatorName]; ok {
			secrets = append(secrets, corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: role.SecretLocation.Namespace,
					Name:      role.SecretLocation.Name,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"azure_client_id":            identity.ClientID,
					"azure_subscription_id":      subscriptionId,
					"azure_tenant_id":            tenantId,
					"azure_region":               region,
					"azure_federated_token_file": azureFederatedTokenFileLocation,
				},
			})
		}
	}

	return secrets, nil
}
