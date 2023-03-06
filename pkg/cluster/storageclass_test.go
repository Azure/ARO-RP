package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestConfigureStorageClass(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mocks      func(kubernetescli *fake.Clientset)
		desID      string
		ocpVersion string
		wantErr    string
		wantNewSC  bool
	}{
		{
			name:       "no disk encryption set provided",
			ocpVersion: version.DefaultInstallStream.Version.String(),
		},
		{
			name:       "disk encryption set provided",
			desID:      "fake-des-id",
			ocpVersion: version.DefaultInstallStream.Version.String(),
			wantNewSC:  true,
		},
		{
			name:       "error getting old default StorageClass",
			desID:      "fake-des-id",
			ocpVersion: version.DefaultInstallStream.Version.String(),
			mocks: func(kubernetescli *fake.Clientset) {
				kubernetescli.PrependReactor("get", "storageclasses", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					if action.(ktesting.GetAction).GetName() != "managed-premium" {
						return false, nil, nil
					}
					return true, nil, errors.New("fake error from get of old StorageClass")
				})
			},
			wantErr: "fake error from get of old StorageClass",
		},
		{
			name:       "error removing default annotation from old StorageClass",
			desID:      "fake-des-id",
			ocpVersion: version.DefaultInstallStream.Version.String(),
			mocks: func(kubernetescli *fake.Clientset) {
				kubernetescli.PrependReactor("update", "storageclasses", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					obj := action.(ktesting.UpdateAction).GetObject().(*storagev1.StorageClass)
					if obj.Name != "managed-premium" {
						return false, nil, nil
					}
					return true, nil, errors.New("fake error from update of old StorageClass")
				})
			},
			wantErr: "fake error from update of old StorageClass",
		},
		{
			name:       "error creating the new default encrypted StorageClass",
			desID:      "fake-des-id",
			ocpVersion: version.DefaultInstallStream.Version.String(),
			mocks: func(kubernetescli *fake.Clientset) {
				kubernetescli.PrependReactor("create", "storageclasses", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					obj := action.(ktesting.CreateAction).GetObject().(*storagev1.StorageClass)
					if obj.Name != "managed-premium-encrypted-cmk" {
						return false, nil, nil
					}
					return true, nil, errors.New("fake error while creating encrypted StorageClass")
				})
			},
			wantErr: "fake error while creating encrypted StorageClass",
		},
		{
			name:       "error creating the new default encrypted StorageClass for 4.11",
			desID:      "fake-des-id",
			ocpVersion: "4.11.16",
			mocks: func(kubernetescli *fake.Clientset) {
				kubernetescli.PrependReactor("create", "storageclasses", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					obj := action.(ktesting.CreateAction).GetObject().(*storagev1.StorageClass)
					if obj.Name != "managed-csi-encrypted-cmk" {
						return false, nil, nil
					}
					return true, nil, errors.New("fake error while creating encrypted StorageClass")
				})
			},
			wantErr: "fake error while creating encrypted StorageClass",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			installVersion, err := version.ParseVersion(tt.ocpVersion)
			if err != nil {
				t.Fatal(err)
			}

			storageClassName := defaultStorageClassName
			encryptedStorageClassName := defaultEncryptedStorageClassName

			if installVersion.V[0] == 4 && installVersion.V[1] == 11 {
				storageClassName = csiStorageClassName
				encryptedStorageClassName = csiEncryptedStorageClassName
			}

			kubernetescli := fake.NewSimpleClientset(
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: storageClassName,
						Annotations: map[string]string{
							"storageclass.kubernetes.io/is-default-class": "true",
						},
					},
				},
			)

			if tt.mocks != nil {
				tt.mocks(kubernetescli)
			}

			m := &manager{
				kubernetescli: kubernetescli,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							WorkerProfiles: []api.WorkerProfile{
								{
									DiskEncryptionSetID: tt.desID,
								},
							},
							ClusterProfile: api.ClusterProfile{
								Version: tt.ocpVersion,
							},
						},
					},
				},
			}

			err = m.configureDefaultStorageClass(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.wantNewSC {
				oldSC, err := kubernetescli.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}

				// Old StorageClass is no longer default
				if oldSC.Annotations["storageclass.kubernetes.io/is-default-class"] != "false" {
					t.Error(oldSC.Annotations["storageclass.kubernetes.io/is-default-class"])
				}

				encryptedSC, err := kubernetescli.StorageV1().StorageClasses().Get(ctx, encryptedStorageClassName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}

				// New StorageClass is default
				if encryptedSC.Annotations["storageclass.kubernetes.io/is-default-class"] != "true" {
					t.Error(encryptedSC.Annotations["storageclass.kubernetes.io/is-default-class"])
				}

				// And has diskEncryptionSetID set to one from worker profile
				if encryptedSC.Parameters["diskEncryptionSetID"] != tt.desID {
					t.Error(encryptedSC.Parameters["diskEncryptionSetID"])
				}
			} else {
				_, err := kubernetescli.StorageV1().StorageClasses().Get(ctx, encryptedStorageClassName, metav1.GetOptions{})
				if !kerrors.IsNotFound(err) {
					t.Error(err)
				}
			}
		})
	}
}
