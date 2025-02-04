package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	mockMsiCertValidityDays = 90
)

var (
	errClusterMsiNotPresentInResponse = errors.New("cluster msi not present in msi credentials response")
)

// ensureClusterMsiCertificate leverages the MSI dataplane module to fetch the MSI's
// backing certificate (if needed) and store the certificate in the cluster MSI key
// vault. It does not concern itself with whether an existing certificate is valid
// or not; that can be left to the certificate refresher component.
func (m *manager) ensureClusterMsiCertificate(ctx context.Context) error {
	secretName := m.clusterMsiSecretName()

	_, err := m.clusterMsiKeyVaultStore.GetSecret(ctx, secretName)
	if err == nil {
		return nil
	} else if err != nil && !azureerrors.IsNotFoundError(err) {
		return err
	}

	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	uaMsiRequest := dataplane.UserAssignedIdentitiesRequest{
		IdentityIds: &[]string{clusterMsiResourceId.String()},
	}

	client, err := m.msiDataplane.NewClient(m.doc.OpenShiftCluster.Identity.IdentityURL)
	if err != nil {
		return err
	}

	msiCredObj, err := client.GetUserAssignedIdentitiesCredentials(ctx, uaMsiRequest)
	if err != nil {
		return err
	}

	now := time.Now()

	var expirationDate time.Time
	if m.env.FeatureIsSet(env.FeatureUseMockMsiRp) {
		expirationDate = now.AddDate(0, 0, mockMsiCertValidityDays)
	} else {
		identity, err := getSingleExplicitIdentity(msiCredObj)
		if err != nil {
			return err
		}
		if identity.NotAfter == nil {
			return errors.New("unable to pull NotAfter from the MSI CredentialsObject")
		}

		// The OpenAPI spec for the MI RP specifies that NotAfter will be "in the format 2017-03-01T14:11:00Z".
		expirationDate, err = time.Parse(time.RFC3339, *identity.NotAfter)
		if err != nil {
			return err
		}
	}

	raw, err := json.Marshal(msiCredObj)
	if err != nil {
		return err
	}

	return m.clusterMsiKeyVaultStore.SetSecret(ctx, secretName, keyvault.SecretSetParameters{
		Value: pointerutils.ToPtr(string(raw)),
		SecretAttributes: &keyvault.SecretAttributes{
			Enabled:   pointerutils.ToPtr(true),
			Expires:   pointerutils.ToPtr(date.UnixTime(expirationDate)),
			NotBefore: pointerutils.ToPtr(date.UnixTime(expirationDate)),
		},
		Tags: nil,
	})
}

// initializeClusterMsiClients intializes any Azure clients that use the cluster
// MSI certificate.
func (m *manager) initializeClusterMsiClients(ctx context.Context) error {
	secretName := m.clusterMsiSecretName()

	kvSecretResponse, err := m.clusterMsiKeyVaultStore.GetSecret(ctx, secretName)
	if err != nil {
		return err
	}

	if kvSecretResponse.Value == nil {
		return fmt.Errorf("secret %q in keyvault missing value", secretName)
	}

	var kvSecret dataplane.ManagedIdentityCredentials
	if err := json.Unmarshal([]byte(*kvSecretResponse.Value), &kvSecret); err != nil {
		return err
	}

	cloud, err := m.env.Environment().CloudNameForMsiDataplane()
	if err != nil {
		return err
	}

	msiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	var azureCred azcore.TokenCredential
	if kvSecret.ExplicitIdentities != nil {
		for _, identity := range *kvSecret.ExplicitIdentities {
			if identity.ResourceId != nil && *identity.ResourceId == msiResourceId.String() {
				var err error
				azureCred, err = dataplane.GetCredential(cloud, identity)
				if err != nil {
					return fmt.Errorf("failed to get credential for msi identity %q: %v", msiResourceId, err)
				}
			}
		}
	}
	if azureCred == nil {
		return fmt.Errorf("managed identity credential missing user-assigned identity %q", msiResourceId)
	}

	// Note that we are assuming that all of the platform MIs are in the same subscription as the ARO resource.
	subId := m.subscriptionDoc.ID
	clientOptions := m.env.Environment().ArmClientOptions()
	clusterMsiFederatedIdentityCredentials, err := armmsi.NewFederatedIdentityCredentialsClient(subId, azureCred, clientOptions)
	if err != nil {
		return err
	}

	userAssignedIdentities, err := armmsi.NewUserAssignedIdentitiesClient(subId, azureCred, clientOptions)
	if err != nil {
		return err
	}

	m.clusterMsiFederatedIdentityCredentials = clusterMsiFederatedIdentityCredentials
	m.userAssignedIdentities = userAssignedIdentities
	return nil
}

// clusterMsiSecretName returns the name to store the cluster MSI certificate under in
// the cluster MSI key vault.
func (m *manager) clusterMsiSecretName() string {
	return dataplane.ManagedIdentityCredentialsStoragePrefix + m.doc.ID
}

func (m *manager) clusterIdentityIDs(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return fmt.Errorf("clusterIdentityIDs called for CSP cluster")
	}

	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	uaMsiRequest := dataplane.UserAssignedIdentitiesRequest{
		IdentityIDs: []*string{ptr.To(clusterMsiResourceId.String())},
	}

	client, err := m.msiDataplane.NewClient(m.doc.OpenShiftCluster.Identity.IdentityURL)
	if err != nil {
		return err
	}

	msiCredObj, err := client.GetUserAssignedIdentitiesCredentials(ctx, uaMsiRequest)
	if err != nil {
		return err
	}

	identity, err := getSingleExplicitIdentity(msiCredObj)
	if err != nil {
		return err
	}
	if identity.ClientId == nil || identity.ObjectId == nil {
		return fmt.Errorf("unable to pull clientID and objectID from the MSI CredentialsObject")
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// we iterate through the existing identities to find the identity matching
		// the expected resourceID with casefolding, to ensure we preserve the
		// passed-in casing on IDs even if it may be incorrect
		for k, v := range doc.OpenShiftCluster.Identity.UserAssignedIdentities {
			if strings.EqualFold(k, clusterMsiResourceId.String()) {
				v.ClientID = *identity.ClientId
				v.PrincipalID = *identity.ObjectId

				doc.OpenShiftCluster.Identity.UserAssignedIdentities[k] = v
				return nil
			}
		}

		return fmt.Errorf("no entries found matching clusterMsiResourceId")
	})

	return err
}

// We expect the GetUserAssignedIdentities request to only ever be made for one identity
// at a time (the cluster MSI) and thus we expect the response to only contain a single
// identity's details.
func getSingleExplicitIdentity(msiCredObj *dataplane.ManagedIdentityCredentials) (dataplane.UserAssignedIdentityCredentials, error) {
	if msiCredObj.ExplicitIdentities == nil ||
		len(*msiCredObj.ExplicitIdentities) == 0 {
		return dataplane.UserAssignedIdentityCredentials{}, errClusterMsiNotPresentInResponse
	}

	return (*msiCredObj.ExplicitIdentities)[0], nil
}

// fixupClusterMsiTenantID repopulates the cluster MSI's tenant ID in the cluster doc by
// getting it from the subscription doc. Note that we are assuming that the MSI is in the
// same tenant as the cluster.
func (m *manager) fixupClusterMsiTenantID(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Identity.TenantID = m.subscriptionDoc.Subscription.Properties.TenantID
		return nil
	})

	return err
}
