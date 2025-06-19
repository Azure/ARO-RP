package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	defaultStorageClassName          = "managed-premium"
	defaultEncryptedStorageClassName = "managed-premium-encrypted-cmk"
	defaultProvisioner               = "kubernetes.io/azure-disk"
	csiStorageClassName              = "managed-csi"
	csiEncryptedStorageClassName     = "managed-csi-encrypted-cmk"
	csiProvisioner                   = "disk.csi.azure.com"
	fileCSIProvisioner               = "file.csi.azure.com"
)

// configureDefaultStorageClass replaces default storage class provided by OCP with
// a new one which uses disk encryption set (if one supplied by a customer).
func (m *manager) configureDefaultStorageClass(ctx context.Context) error {
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)
	workerDiskEncryptionSetID := workerProfiles[0].DiskEncryptionSetID

	if workerDiskEncryptionSetID == "" {
		return nil
	}

	installVersion, err := version.ParseVersion(m.doc.OpenShiftCluster.Properties.ClusterProfile.Version)
	if err != nil {
		return err
	}

	storageClassName := defaultStorageClassName
	encryptedStorageClassName := defaultEncryptedStorageClassName
	provisioner := defaultProvisioner

	// OpenShift 4.11 and above use the CSI storageclasses
	if installVersion.V[0] >= 4 && installVersion.V[1] >= 11 {
		storageClassName = csiStorageClassName
		encryptedStorageClassName = csiEncryptedStorageClassName
		provisioner = csiProvisioner
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		oldSC, err := m.kubernetescli.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if oldSC.Annotations == nil {
			oldSC.Annotations = map[string]string{}
		}
		oldSC.Annotations["storageclass.kubernetes.io/is-default-class"] = "false"
		_, err = m.kubernetescli.StorageV1().StorageClasses().Update(ctx, oldSC, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		encryptedSC := newEncryptedStorageClass(workerDiskEncryptionSetID, encryptedStorageClassName, provisioner)
		_, err = m.kubernetescli.StorageV1().StorageClasses().Create(ctx, encryptedSC, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		return nil
	})
}

// For Workload Identity clusters, the default azurefile-csi storage class must be removed
// By default the azurefile-csi storage class can choose Cluster Storage Account for file share creation
// Since Cluster Storage Account will have shared key disabled, file share will not be supported in this scenario.
func (m *manager) removeAzureFileCSIStorageClass(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		csiDriver, err := m.operatorcli.OperatorV1().ClusterCSIDrivers().Get(ctx, fileCSIProvisioner, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		csiDriver.Spec.StorageClassState = "Removed"
		_, err = m.operatorcli.OperatorV1().ClusterCSIDrivers().Update(ctx, csiDriver, metav1.UpdateOptions{})
		return err
	})
}

func newEncryptedStorageClass(diskEncryptionSetID, encryptedStorageClassName, provisioner string) *storagev1.StorageClass {
	volumeBindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: encryptedStorageClassName,
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner:          provisioner,
		VolumeBindingMode:    &volumeBindingMode,
		AllowVolumeExpansion: to.Ptr(true),
		ReclaimPolicy:        &reclaimPolicy,
		Parameters: map[string]string{
			"kind":                "Managed",
			"storageaccounttype":  "Premium_LRS",
			"diskEncryptionSetID": diskEncryptionSetID,
		},
	}
}
