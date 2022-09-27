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
)

const (
	defaultStorageClassName          = "managed-csi"
	defaultEncryptedStorageClassName = "managed-csi-encrypted-cmk"
)

// configureDefaultStorageClass replaces default storage class provided by OCP with
// a new one which uses disk encryption set (if one supplied by a customer).
func (m *manager) configureDefaultStorageClass(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskEncryptionSetID == "" {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		oldSC, err := m.kubernetescli.StorageV1().StorageClasses().Get(ctx, defaultStorageClassName, metav1.GetOptions{})
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

		encryptedSC := newEncryptedStorageClass(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskEncryptionSetID)
		_, err = m.kubernetescli.StorageV1().StorageClasses().Create(ctx, encryptedSC, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		return nil
	})
}

func newEncryptedStorageClass(diskEncryptionSetID string) *storagev1.StorageClass {
	volumeBindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultEncryptedStorageClassName,
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner:       "disk.csi.azure.com",
		VolumeBindingMode: &volumeBindingMode,
		ReclaimPolicy:     &reclaimPolicy,
		Parameters: map[string]string{
			"storageAccountType":  "Premium_LRS",
			"diskEncryptionSetID": diskEncryptionSetID,
		},
	}
}
