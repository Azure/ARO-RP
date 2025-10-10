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
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

var (
	errClusterMsiNotPresentInResponse = errors.New("cluster msi not present in msi credentials response")
)

// ensureClusterMsiCertificate leverages the MSI dataplane module to fetch the MSI's
// backing certificate (if needed) and store the certificate in the cluster MSI key
// vault. If the certificate stored in keyvault is eligible for renewal, the
// certificate is empty or the certificate is for a different identity, this
// function will request and persist a new certificate.
func (m *manager) ensureClusterMsiCertificate(ctx context.Context, now time.Time) error {
	secretName := dataplane.IdentifierForManagedIdentityCredentials(m.doc.ID)

	existingMsiCertificate, err := m.clusterMsiKeyVaultStore.GetSecret(ctx, secretName, "", nil)
	if err != nil && !azureerrors.IsNotFoundError(err) {
		return err
	}

	// If the secret exists, we need to decide if it should be replaced.
	if err == nil {
		replace, err := m.shouldReplaceMSICertificate(&existingMsiCertificate, now)
		if err != nil {
			return err
		}
		if !replace {
			// The existing certificate is valid, so we're done.
			return nil
		}
	}
	// If we reach this point, it's because the secret was not found, or it was found but is invalid/expired.
	// In either case, we need to create a new one.

	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	uaMsiRequest := dataplane.UserAssignedIdentitiesRequest{
		IdentityIDs: []string{clusterMsiResourceId.String()},
	}

	client, err := m.msiDataplane.NewClient(m.doc.OpenShiftCluster.Identity.IdentityURL)
	if err != nil {
		return err
	}

	msiCredObj, err := client.GetUserAssignedIdentitiesCredentials(ctx, uaMsiRequest)
	if err != nil {
		return err
	}

	name, parameters, err := dataplane.FormatManagedIdentityCredentialsForStorage(m.doc.ID, *msiCredObj)
	if err != nil {
		return fmt.Errorf("failed to format managed identity credentials for storage: %w", err)
	}

	_, err = m.clusterMsiKeyVaultStore.SetSecret(ctx, name, parameters, nil)
	return err
}

func (m *manager) shouldReplaceMSICertificate(cert *azsecrets.GetSecretResponse, now time.Time) (bool, error) {
	if cert.Attributes == nil || cert.Value == nil {
		return true, nil
	}

	var keyvaultCredentials dataplane.ManagedIdentityCredentials
	if err := json.Unmarshal([]byte(*cert.Value), &keyvaultCredentials); err != nil {
		return false, err
	}

	if len(keyvaultCredentials.ExplicitIdentities) == 0 {
		return true, nil
	}

	// Check if the secret is for a different identity (e.g., after a cluster update).
	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return false, err
	}
	if keyvaultCredentials.ExplicitIdentities[0].ResourceID == nil ||
		*keyvaultCredentials.ExplicitIdentities[0].ResourceID != clusterMsiResourceId.String() {
		return true, nil
	}

	// Check if the certificate is within its renewal window.
	// In the future, certificate refreshing will be handled by the Certificate Refresher. For now, handle it here.
	return m.needsRefresh(cert, now)
}

// https://eng.ms/docs/products/arm/rbac/managed_identities/msionboardingcertificaterotation
// The cert is eligible to be refreshed after the 46 day mark, and expires at 90 days
// This is subject to change and docs can be untrustworthy, so use the keyvault tags to determine validity
func (m *manager) needsRefresh(item *azsecrets.GetSecretResponse, now time.Time) (bool, error) {
	if item.Tags == nil {
		return false, fmt.Errorf("secret tags are nil")
	}

	var renewAfter, cannotRenewAfter time.Time

	tagsToParse := map[string]*time.Time{
		dataplane.RenewAfterKeyVaultTag:       &renewAfter,
		dataplane.CannotRenewAfterKeyVaultTag: &cannotRenewAfter,
	}

	for tagKey, timeVarPtr := range tagsToParse {
		valuePtr, ok := item.Tags[tagKey]
		if !ok || valuePtr == nil {
			return false, fmt.Errorf("missing or invalid tag: %s", tagKey)
		}

		parsedTime, err := time.Parse(time.RFC3339, *valuePtr)
		if err != nil {
			return false, fmt.Errorf("invalid time format for tag %s: %w", tagKey, err)
		}

		*timeVarPtr = parsedTime
	}

	inRenewalWindow := !renewAfter.After(now) && !now.After(cannotRenewAfter)

	return inRenewalWindow, nil
}

// initializeClusterMsiClients intializes any Azure clients that use the cluster
// MSI certificate.
func (m *manager) initializeClusterMsiClients(ctx context.Context) error {
	secretName := dataplane.IdentifierForManagedIdentityCredentials(m.doc.ID)

	kvSecretResponse, err := m.clusterMsiKeyVaultStore.GetSecret(ctx, secretName, "", nil)
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

	msiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	var azureCred azcore.TokenCredential
	for _, identity := range kvSecret.ExplicitIdentities {
		if identity.ResourceID != nil && strings.EqualFold(*identity.ResourceID, msiResourceId.String()) {
			var err error
			azureCred, err = dataplane.GetCredential(m.env.Environment().AzureClientOptions(), identity)
			if err != nil {
				return fmt.Errorf("failed to get credential for msi identity %q: %v", msiResourceId, err)
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

func (m *manager) clusterIdentityIDs(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return fmt.Errorf("clusterIdentityIDs called for CSP cluster")
	}

	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	uaMsiRequest := dataplane.UserAssignedIdentitiesRequest{
		IdentityIDs: []string{clusterMsiResourceId.String()},
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
	if identity.ClientID == nil || identity.ObjectID == nil {
		return fmt.Errorf("unable to pull clientID and objectID from the MSI CredentialsObject")
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// we iterate through the existing identities to find the identity matching
		// the expected resourceID with casefolding, to ensure we preserve the
		// passed-in casing on IDs even if it may be incorrect
		for k, v := range doc.OpenShiftCluster.Identity.UserAssignedIdentities {
			if strings.EqualFold(k, clusterMsiResourceId.String()) {
				v.ClientID = *identity.ClientID
				v.PrincipalID = *identity.ObjectID

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
	if len(msiCredObj.ExplicitIdentities) == 0 {
		return dataplane.UserAssignedIdentityCredentials{}, errClusterMsiNotPresentInResponse
	}

	return msiCredObj.ExplicitIdentities[0], nil
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
