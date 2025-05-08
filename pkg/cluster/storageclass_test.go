package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/version"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestConfigureStorageClass(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mocks      func(kubernetescli *fake.Clientset)
		desID      string
		wpStatus   bool
		ocpVersion string
		wantErr    string
		wantNewSC  bool
	}{
		{
			name:       "no disk encryption set provided",
			ocpVersion: "4.10.40",
		},
		{
			name:       "disk encryption set provided",
			desID:      "fake-des-id",
			ocpVersion: "4.10.40",
			wantNewSC:  true,
		},
		{
			name:       "Use disk encryption set provided in enriched worker profile",
			desID:      "fake-des-id",
			wpStatus:   true,
			ocpVersion: "4.10.40",
			wantNewSC:  true,
		},
		{
			name:       "error getting old default StorageClass",
			desID:      "fake-des-id",
			ocpVersion: "4.10.40",
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
			ocpVersion: "4.10.40",
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
			ocpVersion: "4.10.40",
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

			if tt.wpStatus {
				m.doc.OpenShiftCluster.Properties.WorkerProfiles = nil
				m.doc.OpenShiftCluster.Properties.WorkerProfilesStatus = []api.WorkerProfile{
					{
						DiskEncryptionSetID: tt.desID,
					},
				}
			}

			err = m.configureDefaultStorageClass(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

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

func TestRemoveAzureFileCSIStorageClass(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name        string
		operatorcli func() *operatorfake.Clientset
		doc         api.OpenShiftCluster
		wantRemoved bool
	}

	for _, tt := range []*test{
		{
			name: "noop - Cluster Service Principal Cluster",
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecret",
					},
				},
			},
		},
		{
			name: "noop - fileCSIProvisioner doesn't exist",
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			operatorcli: func() *operatorfake.Clientset {
				return operatorfake.NewSimpleClientset(
					&operatorv1.ClusterCSIDriver{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nonFileCSIProvisioner",
						},
					},
				)
			},
		},
		{
			name: "Pass - updated fileCSIProvisioner",
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			operatorcli: func() *operatorfake.Clientset {
				return operatorfake.NewSimpleClientset(
					&operatorv1.ClusterCSIDriver{
						ObjectMeta: metav1.ObjectMeta{
							Name: fileCSIProvisioner,
						},
					},
				)
			},
			wantRemoved: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &tt.doc,
				},
			}

			if tt.operatorcli != nil {
				m.operatorcli = tt.operatorcli()
			}

			err := m.removeAzureFileCSIStorageClass(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantRemoved {
				clusterCSIDriver, err := m.operatorcli.OperatorV1().ClusterCSIDrivers().Get(ctx, fileCSIProvisioner, metav1.GetOptions{})
				if err != nil || clusterCSIDriver == nil {
					t.Fatal("Expected clusterCSIDriver but returned error")
				}
				assert.Equal(t, clusterCSIDriver.Spec.StorageClassState, operatorv1.StorageClassStateName("Removed"))
			}
		})
	}
}
