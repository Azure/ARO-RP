package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/Azure/msi-dataplane/pkg/store"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
)

const (
	mockMsiCertValidityDays = 90
)

// ensureClusterMsiCertificate leverages the MSI dataplane module to fetch the MSI's
// backing certificate (if needed) and store the certificate in the cluster MSI key
// vault. It does not concern itself with whether an existing certificate is valid
// or not; that can be left to the certificate refresher component.
func (m *manager) ensureClusterMsiCertificate(ctx context.Context) error {
	secretName, err := m.clusterMsiSecretName()
	if err != nil {
		return err
	}

	_, err = m.clusterMsiKeyVaultStore.GetCredentialsObject(ctx, secretName)
	if err == nil {
		return nil
	} else if azcoreErr, ok := err.(*azcore.ResponseError); !ok || azcoreErr.StatusCode != http.StatusNotFound {
		return err
	}

	clusterMsiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	uaMsiRequest := dataplane.UserAssignedMSIRequest{
		IdentityURL: m.doc.OpenShiftCluster.Identity.IdentityURL,
		ResourceIDs: []string{clusterMsiResourceId.String()},
		TenantID:    m.doc.OpenShiftCluster.Identity.TenantID,
	}

	msiCredObj, err := m.msiDataplane.GetUserAssignedIdentities(ctx, uaMsiRequest)
	if err != nil {
		return err
	}

	now := time.Now()

	var expirationDate time.Time
	if m.env.FeatureIsSet(env.FeatureUseMockMsiRp) {
		expirationDate = now.AddDate(0, 0, mockMsiCertValidityDays)
	} else {
		if msiCredObj.CredentialsObject.ExplicitIdentities == nil || len(msiCredObj.CredentialsObject.ExplicitIdentities) == 0 || msiCredObj.CredentialsObject.ExplicitIdentities[0] == nil || msiCredObj.CredentialsObject.ExplicitIdentities[0].NotAfter == nil {
			return errors.New("unable to pull NotAfter from the MSI CredentialsObject")
		}

		// The swagger API spec for the MI RP specifies that NotAfter will be "in the format 2017-03-01T14:11:00Z".
		expirationDate, err = time.Parse(time.RFC3339, *msiCredObj.CredentialsObject.ExplicitIdentities[0].NotAfter)
		if err != nil {
			return err
		}
	}

	secretProperties := store.SecretProperties{
		Enabled:   true,
		Expires:   expirationDate,
		Name:      secretName,
		NotBefore: now,
	}

	return m.clusterMsiKeyVaultStore.SetCredentialsObject(ctx, secretProperties, msiCredObj.CredentialsObject)
}

// initializeClusterMsiClients intializes any Azure clients that use the cluster
// MSI certificate.
func (m *manager) initializeClusterMsiClients(ctx context.Context) error {
	secretName, err := m.clusterMsiSecretName()
	if err != nil {
		return err
	}

	kvSecret, err := m.clusterMsiKeyVaultStore.GetCredentialsObject(ctx, secretName)
	if err != nil {
		return err
	}

	cloud, err := m.env.Environment().CloudNameForMsiDataplane()
	if err != nil {
		return err
	}

	uaIdentities, err := dataplane.NewUserAssignedIdentities(kvSecret.CredentialsObject, cloud)
	if err != nil {
		return err
	}

	msiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return err
	}

	azureCred, err := uaIdentities.GetCredential(msiResourceId.String())
	if err != nil {
		return err
	}

	// Note that we are assuming that all of the platform MIs are in the same subscription.
	pwiResourceId, err := arm.ParseResourceID(m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[0].ResourceID)
	if err != nil {
		return err
	}

	subId := pwiResourceId.SubscriptionID
	clientOptions := m.env.Environment().ArmClientOptions()
	clusterMsiFederatedIdentityCredentials, err := armmsi.NewFederatedIdentityCredentialsClient(subId, azureCred, clientOptions)
	if err != nil {
		return err
	}

	m.clusterMsiFederatedIdentityCredentials = clusterMsiFederatedIdentityCredentials
	return nil
}

// clusterMsiSecretName returns the name to store the cluster MSI certificate under in
// the cluster MSI key vault.
func (m *manager) clusterMsiSecretName() (string, error) {
	clusterMsi, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", m.doc.ID, clusterMsi.Name), nil
}
