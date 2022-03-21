package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

// Test reconcile function
func TestImageConfigReconciler(t *testing.T) {
	type test struct {
		name                string
		arocli              aroclient.Interface
		configcli           configclient.Interface
		wantRegistrySources configv1.RegistrySources
		wantErr             string
	}

	for _, tt := range []*test{
		{
			name: "Feature Flag disabled, no action",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(false),
					},
					Location: "eastus",
				},
			}),
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "Image config registry source is empty, no action",
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
			}),
			wantRegistrySources: configv1.RegistrySources{},
		},
		{
			name: "allowedRegistries exists with duplicates, function should appropriately add registries",
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
							"arointsvc.azurecr.io",
							"arointsvc.azurecr.io",
						},
					},
				},
			}),
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
					"arointsvc.azurecr.io",
					"arosvc.eastus.data.azurecr.io",
				},
			},
		},
		{
			name: "blockedRegistries exists, function should delete registries",
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						BlockedRegistries: []string{
							"quay.io",
							"arointsvc.azurecr.io",
							"arosvc.eastus.data.azurecr.io",
						},
					},
				},
			}),
			wantRegistrySources: configv1.RegistrySources{
				BlockedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "AZEnvironment is unset, no action",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(true),
					},
				},
			}),
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "Both AllowedRegistries and BlockedRegistries are present, function should fail silently and not requeue",
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						BlockedRegistries: []string{
							"arointsvc.azurecr.io",
							"arosvc.eastus.data.azurecr.io",
						},
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantRegistrySources: configv1.RegistrySources{
				BlockedRegistries: []string{
					"arointsvc.azurecr.io",
					"arosvc.eastus.data.azurecr.io",
				},
				AllowedRegistries: []string{
					"quay.io",
				},
			},
			wantErr: `both AllowedRegistries and BlockedRegistries are present`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var arocli aroclient.Interface
			if tt.arocli != nil {
				arocli = tt.arocli
			} else {
				arocli = arofake.NewSimpleClientset(&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
					Spec: arov1alpha1.ClusterSpec{
						ACRDomain:     "arointsvc.azurecr.io",
						AZEnvironment: azureclient.PublicCloud.Environment.Name,
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: strconv.FormatBool(true),
						},
						Location: "eastus",
					},
				})
			}

			r := &Reconciler{
				arocli:    arocli,
				configcli: tt.configcli,
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
			imgcfg, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(imgcfg.Spec.RegistrySources, tt.wantRegistrySources) {
				t.Error(cmp.Diff(imgcfg.Spec.RegistrySources, tt.wantRegistrySources))
			}
		})
	}
}

func TestGetCloudAwareRegistries(t *testing.T) {
	type test struct {
		name       string
		instance   *arov1alpha1.Cluster
		wantResult []string
		wantErr    string
	}

	for _, tt := range []*test{
		{
			name: "public cloud",
			instance: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					Location:      "eastus",
				},
			},
			wantResult: []string{"arointsvc.azurecr.io", "arosvc.eastus.data.azurecr.io"},
		},
		{
			name: "us gov cloud",
			instance: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.us",
					AZEnvironment: azureclient.USGovernmentCloud.Environment.Name,
					Location:      "eastus",
				},
			},
			wantResult: []string{"arointsvc.azurecr.us", "arosvc.eastus.data.azurecr.us"},
		},
		{
			name: "unsupported cloud",
			instance: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: "FakeCloud",
					Location:      "eastus",
				},
			},
			wantErr: "cloud environment FakeCloud is not supported",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getCloudAwareRegistries(tt.instance)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if !reflect.DeepEqual(result, tt.wantResult) {
				t.Error(cmp.Diff(result, tt.wantResult))
			}
		})
	}
}
