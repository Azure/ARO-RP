package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	defaultStorageClassName          = "managed-premium"
	defaultEncryptedStorageClassName = "managed-premium-encrypted-cmk"
	defaultProvisioner               = "kubernetes.io/azure-disk"
	csiStorageClassName              = "managed-csi"
	csiEncryptedStorageClassName     = "managed-csi-encrypted-cmk"
	csiProvisioner                   = "disk.csi.azure.com"
)

// configureDefaultStorageClass replaces default storage class provided by OCP with
// a new one which uses disk encryption set (if one supplied by a customer).
func (m *manager) configureDefaultStorageClass(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskEncryptionSetID == "" {
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

		encryptedSC := newEncryptedStorageClass(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskEncryptionSetID, encryptedStorageClassName, provisioner)
		_, err = m.kubernetescli.StorageV1().StorageClasses().Create(ctx, encryptedSC, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		return nil
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
		AllowVolumeExpansion: to.BoolPtr(true),
		ReclaimPolicy:        &reclaimPolicy,
		Parameters: map[string]string{
			"kind":                "Managed",
			"storageaccounttype":  "Premium_LRS",
			"diskEncryptionSetID": diskEncryptionSetID,
		},
	}
}
