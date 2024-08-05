package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/Azure/msi-dataplane/pkg/store"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
)

const (
	msiCertValidityDays = 90
)

// ensureClusterMsiCertificate leverages the MSI dataplane module to create a new MSI
// certificate (if needed) and store the certificate in the cluster MSI key vault. It
// does not concern itself with whether an existing certificate is valid or not; that
// can be left to the certificate refresher component.
func (m *manager) ensureClusterMsiCertificate(ctx context.Context) error {
	secretName, err := m.clusterMsiSecretName()
	if err != nil {
		return err
	}

	_, err = m.clusterMsiKeyVaultStore.GetCredentialsObject(ctx, secretName)
	if err == nil {
		return nil
	} else if azcoreErr, ok := err.(*azcore.ResponseError); !ok || azcoreErr.StatusCode != 404 {
		return err
	}

	clusterMsiResourceId, err := m.clusterMsiResourceId()
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
	expirationDate := now.AddDate(0, 0, msiCertValidityDays)
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

	// This code assumes that there is only one cluster MSI and will have to be
	// refactored if we ever use more than one.
	if kvSecret.CredentialsObject.ExplicitIdentities == nil {
		return errors.New("found nil ExplicitIdentities in cluster MSI CredentialsObject")
	} else if len(kvSecret.CredentialsObject.ExplicitIdentities) == 0 {
		return errors.New("found empty ExplicitIdentities in cluster MSI CredentialsObject")
	} else if len(kvSecret.CredentialsObject.ExplicitIdentities) > 1 {
		m.log.Warning("unexpectedly found more than one entry in ExplicitIdentities in cluster MSI CredentialsObject; will attempt to instantiate a cluster MSI credential using the first one")
	}

	if kvSecret.CredentialsObject.ExplicitIdentities[0].ClientID == nil {
		return errors.New("found nil ClientID while parsing cluster MSI CredentialsObject")
	}
	if kvSecret.CredentialsObject.ExplicitIdentities[0].TenantID == nil {
		return errors.New("found nil TenantID while parsing cluster MSI CredentialsObject")
	}
	if kvSecret.CredentialsObject.ExplicitIdentities[0].ClientSecret == nil {
		return errors.New("found nil ClientSecret while parsing cluster MSI CredentialsObject")
	}

	clientId := *kvSecret.CredentialsObject.ExplicitIdentities[0].ClientID
	tenantId := *kvSecret.CredentialsObject.ExplicitIdentities[0].TenantID
	certData := *kvSecret.CredentialsObject.ExplicitIdentities[0].ClientSecret

	decodedCertData, err := base64.StdEncoding.DecodeString(certData)
	if err != nil {
		return err
	}

	certs, key, err := azidentity.ParseCertificates(decodedCertData, nil)
	if err != nil {
		return err
	}

	cred, err := azidentity.NewClientCertificateCredential(tenantId, clientId, certs, key, m.env.Environment().ClientCertificateCredentialOptions())
	if err != nil {
		return err
	}

	// Note that we are assuming that all of the platform MIs are in the same subscription.
	resourceId, err := arm.ParseResourceID(m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[0].ResourceID)
	if err != nil {
		return err
	}

	subId := resourceId.SubscriptionID
	clientOptions := getClientOptions(m.env.Environment())
	clusterMsiFederatedIdentityCredentials, err := armmsi.NewFederatedIdentityCredentialsClient(subId, cred, &clientOptions)
	if err != nil {
		return err
	}

	m.clusterMsiFederatedIdentityCredentials = clusterMsiFederatedIdentityCredentials
	return nil
}

// clusterMsiSecretName returns the name to store the cluster MSI certificate under in
// the cluster MSI key vault.
func (m *manager) clusterMsiSecretName() (string, error) {
	clusterMsi, err := m.clusterMsiResourceId()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", m.doc.ID, clusterMsi.Name), nil
}

// clusterMsiResourceId returns the resource ID of the cluster MSI or an error
// if it encounters an issue while grabbing the resource ID from the cluster
// doc. It is written under the assumption that there is only one cluster MSI
// and will have to be refactored if we ever use more than one.
func (m *manager) clusterMsiResourceId() (*arm.ResourceID, error) {
	var clusterMsi *arm.ResourceID
	if m.doc.OpenShiftCluster.Identity != nil && m.doc.OpenShiftCluster.Identity.UserAssignedIdentities != nil {
		for msiResourceId := range m.doc.OpenShiftCluster.Identity.UserAssignedIdentities {
			var err error
			clusterMsi, err = arm.ParseResourceID(msiResourceId)
			if err != nil {
				return nil, err
			}
		}
	}

	if clusterMsi == nil {
		return nil, errors.New("could not find cluster MSI in cluster doc")
	}

	return clusterMsi, nil
}
