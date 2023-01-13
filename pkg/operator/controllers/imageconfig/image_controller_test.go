package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

// Test reconcile function
func TestImageConfigReconciler(t *testing.T) {
	type test struct {
		name                string
		instance            *arov1alpha1.Cluster
		image               *configv1.Image
		wantRegistrySources configv1.RegistrySources
		wantErr             string
	}

	for _, tt := range []*test{
		{
			name: "Feature Flag disabled, no action",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(false),
					},
					Location: "eastus",
				},
			},
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			},
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "Image config registry source is empty, no action",
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
			},
			wantRegistrySources: configv1.RegistrySources{},
		},
		{
			name: "allowedRegistries exists with duplicates, function should appropriately add registries",
			image: &configv1.Image{
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
			},
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
					"arointsvc.azurecr.io",
					"arointsvc.eastus.data.azurecr.io",
				},
			},
		},
		{
			name: "blockedRegistries exists, function should delete registries",
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						BlockedRegistries: []string{
							"quay.io",
							"arointsvc.azurecr.io",
							"arointsvc.eastus.data.azurecr.io",
						},
					},
				},
			},
			wantRegistrySources: configv1.RegistrySources{
				BlockedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "AZEnvironment is unset, no action",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(true),
					},
				},
			},
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			},
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
				},
			},
		},
		{
			name: "Both AllowedRegistries and BlockedRegistries are present, function should fail silently and not requeue",
			image: &configv1.Image{
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
			},
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
		{
			name: "uses Public Cloud cluster's ACRDomain configuration for both Azure registries",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "fakesvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(true),
					},
					Location: "anyplace",
				},
			},
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{"quay.io"},
					},
				},
			},
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
					"fakesvc.azurecr.io",
					"fakesvc.anyplace.data.azurecr.io",
				},
			},
		},
		{
			name: "uses USGov Cloud cluster's ACRDomain configuration for both Azure registries",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "fakesvc.azurecr.us",
					AZEnvironment: azureclient.USGovernmentCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(true),
					},
					Location: "anyplace",
				},
			},
			image: &configv1.Image{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{"quay.io"},
					},
				},
			},
			wantRegistrySources: configv1.RegistrySources{
				AllowedRegistries: []string{
					"quay.io",
					"fakesvc.azurecr.us",
					"fakesvc.anyplace.data.azurecr.us",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: strconv.FormatBool(true),
					},
					Location: "eastus",
				},
			}
			if tt.instance != nil {
				instance = tt.instance
			}

			clientFake := ctrlfake.NewClientBuilder().WithObjects(instance, tt.image).Build()

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				client: clientFake,
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			imgcfg := &configv1.Image{}
			err = r.client.Get(ctx, types.NamespacedName{Name: request.Name}, imgcfg)
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
					ACRDomain:     "arosvc.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					Location:      "eastus",
				},
			},
			wantResult: []string{"arosvc.azurecr.io", "arosvc.eastus.data.azurecr.io"},
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
			wantResult: []string{"arointsvc.azurecr.us", "arointsvc.eastus.data.azurecr.us"},
		},
		{
			name: "arbitrary name",
			instance: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "fakeacr.azurecr.io",
					AZEnvironment: azureclient.PublicCloud.Environment.Name,
					Location:      "anyplace",
				},
			},
			wantResult: []string{"fakeacr.azurecr.io", "fakeacr.anyplace.data.azurecr.io"},
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
			result, err := GetCloudAwareRegistries(tt.instance)

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
