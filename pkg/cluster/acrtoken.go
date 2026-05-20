package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var ErrNoRegistryProfileFound = errors.New("no registry profile found")

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

func newACRTokenManager(_env env.Interface) (acrtoken.Manager, error) {
	acrR, err := azure.ParseResourceID(_env.ACRResourceID())
	if err != nil {
		return nil, err
	}

	fpCredRPTenant, err := _env.FPNewClientCertificateCredential(_env.TenantID(), nil)
	if err != nil {
		return nil, err
	}

	armRPTokensClient, err := armcontainerregistry.NewTokensClient(acrR.SubscriptionID, fpCredRPTenant, _env.Environment().ArmClientOptions())
	if err != nil {
		return nil, err
	}

	armRPRegistriesClient, err := armcontainerregistry.NewRegistriesClient(acrR.SubscriptionID, fpCredRPTenant, _env.Environment().ArmClientOptions())
	if err != nil {
		return nil, err
	}

	return acrtoken.NewManager(_env, armRPTokensClient, armRPRegistriesClient)
}

func (m *manager) ensureACRToken(ctx context.Context) error {
	if m.env.IsLocalDevelopmentMode() {
		return nil
	}

	token, err := newACRTokenManager(m.env)
	if err != nil {
		return err
	}

	rp := m.doc.OpenShiftCluster.GetRegistryProfile(m.env.ACRDomain())
	if rp == nil {
		// 1. choose a name and establish the intent to create a token with
		// that name
		rp = token.NewRegistryProfile()

		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.PutRegistryProfile(rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	if rp.Password == "" {
		// 2. ensure a token with the chosen name exists, generate a
		// password for it and store it in the database
		password, err := token.EnsureTokenAndPassword(ctx, rp)
		if err != nil {
			return err
		}
		currentTime := m.env.Now().UTC()
		rp.Password = api.SecureString(password)
		rp.IssueDate = &currentTime

		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.PutRegistryProfile(rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) rotateACRTokenPassword(ctx context.Context) error {
	// we do not want to rotate tokens in local development
	if m.env.IsLocalDevelopmentMode() || m.env.IsCI() {
		return nil
	}

	token, err := newACRTokenManager(m.env)
	if err != nil {
		return err
	}

	updateDB := func(ctx context.Context, oscdm database.OpenShiftClusterDocumentMutator) (*api.OpenShiftClusterDocument, error) {
		return m.db.PatchWithLease(ctx, m.doc.Key, oscdm)
	}

	return RotateACRToken(ctx, m.env, m.log, m.ch, m.doc, token, updateDB, false)
}

func RotateACRToken(ctx context.Context, env env.Interface, log *logrus.Entry, ch clienthelper.Interface, doc *api.OpenShiftClusterDocument, token acrtoken.Manager, updateDB database.OpenShiftClusterDocumentMutatorRunner, force bool) error {
	registryProfile := doc.OpenShiftCluster.GetRegistryProfile(env.ACRDomain())
	if registryProfile == nil {
		return ErrNoRegistryProfileFound
	}

	shouldRotate, _, durationUntilRotate, validityRemaining := acrtoken.ShouldRotateToken(env, registryProfile)
	log.Infof("token has %s validity remaining, should rotate in %s", validityRemaining.String(), durationUntilRotate.String())
	if !shouldRotate && !force {
		return nil
	} else if !shouldRotate && force {
		log.Infof("force rotating token before rotation period")
	}

	log.Infof("rotating ACR token")
	err := token.RotateTokenPassword(ctx, registryProfile)
	if err != nil {
		return err
	}

	doc, err = updateDB(ctx, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.PutRegistryProfile(registryProfile)
		return nil
	})
	if err != nil {
		return err
	}

	// update cluster pull secret in openshift-azure-operator namespace
	// secret is stored as a .dockerconfigjson string in the .dockerconfigjson key
	encodedDockerConfigJson, _, err := pullsecret.SetRegistryProfiles("", registryProfile)
	if err != nil {
		return err
	}

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operator.SecretName,
			Namespace: operator.Namespace,
		},
		Data: make(map[string][]byte),
	}
	pullSecret.Data[corev1.DockerConfigJsonKey] = []byte(encodedDockerConfigJson)

	err = ch.Update(ctx, pullSecret)
	if err != nil {
		return err
	}

	return retryOperation(func() error {
		return rotateOpenShiftConfigSecret(ctx, log, ch, pullSecret.Data[corev1.DockerConfigJsonKey])
	})
}

func rotateOpenShiftConfigSecret(ctx context.Context, log *logrus.Entry, ch clienthelper.Interface, encodedDockerConfigJson []byte) error {
	openshiftConfigSecret := &corev1.Secret{}

	err := ch.GetOne(ctx, pullSecretName, openshiftConfigSecret)
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	// by default, we create a patch with only the rotated acr token
	applyConfiguration := v1.Secret(pullSecretName.Name, pullSecretName.Namespace).
		WithData(map[string][]byte{corev1.DockerConfigJsonKey: encodedDockerConfigJson}).
		WithType(corev1.SecretTypeDockerConfigJson)

	recreationOfSecretRequired := openshiftConfigSecret == nil ||
		(openshiftConfigSecret.Type != corev1.SecretTypeDockerConfigJson || openshiftConfigSecret.Data == nil) ||
		(openshiftConfigSecret.Immutable != nil && *openshiftConfigSecret.Immutable)

	if recreationOfSecretRequired {
		err := retryOperation(func() error {
			return ch.EnsureDeleted(ctx, metav1.SchemeGroupVersion.WithKind("Secret"), pullSecretName)
		})
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}
	}

	// attempt to merge the data
	if openshiftConfigSecret != nil && openshiftConfigSecret.Data != nil {
		previousConfigData, previousConfigDataExists := openshiftConfigSecret.Data[corev1.DockerConfigJsonKey]
		if previousConfigDataExists {
			mergedPullSecretData, _, err := pullsecret.Merge(string(previousConfigData), string(encodedDockerConfigJson))
			if err == nil {
				applyConfiguration.Data[corev1.DockerConfigJsonKey] = []byte(mergedPullSecretData)
			} else {
				log.Error("Could not merge openshift config pull secret, overriding with new acr token", err)
			}
		}
	}

	d, err := json.Marshal(applyConfiguration)
	if err != nil {
		return err
	}

	return retryOperation(func() error {
		return ch.Patch(ctx, openshiftConfigSecret, client.RawPatch(types.ApplyPatchType, d), &client.PatchOptions{FieldManager: "aro-rp", Force: pointerutils.ToPtr(true)})
	})
}

func retryOperation(retryable func() error) error {
	return retry.OnError(wait.Backoff{
		Steps:    10,
		Duration: 2 * time.Second,
	}, func(err error) bool {
		return kerrors.IsBadRequest(err) || kerrors.IsInternalError(err) || kerrors.IsServerTimeout(err) || kerrors.IsConflict(err)
	}, retryable)
}
