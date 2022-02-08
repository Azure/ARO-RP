package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

// fake arocli
var (
	imageConfigMetadata = metav1.ObjectMeta{Name: "cluster"}
	arocli              = arofake.NewSimpleClientset(&arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ACRDomain:     "arointsvc.azurecr.io",
			AZEnvironment: "AzurePublicCloud",
			OperatorFlags: arov1alpha1.OperatorFlags{
				ENABLED: strconv.FormatBool(true),
			},
			Location: "eastus",
		},
	})
)

// Test reconcile function
func TestImageConfigReconciler(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	type test struct {
		name       string
		arocli     aroclient.Interface
		configcli  configclient.Interface
		wantConfig string
		wantErr    string
	}

	for _, tt := range []*test{
		{
			name: "Feature Flag disabled, no action",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.io",
					AZEnvironment: "AzurePublicCloud",
					OperatorFlags: arov1alpha1.OperatorFlags{
						ENABLED: strconv.FormatBool(false),
					},
					Location: "eastus",
				},
			}),
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantConfig: `["quay.io"]`,
		},
		{
			name:   "Image config registry source is empty, no action",
			arocli: arocli,
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
			}),
			wantConfig: `null`,
		},
		{
			name:   "allowedRegistries exists with duplicates, function should appropriately add registries",
			arocli: arocli,
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
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
			wantConfig: `["arointsvc.azurecr.io","arosvc.eastus.data.azurecr.io","quay.io"]`,
		},
		{
			name:   "blockedRegistries exists, function should delete registries",
			arocli: arocli,
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						BlockedRegistries: []string{
							"quay.io", "arointsvc.azurecr.io", "arosvc.eastus.data.azurecr.io",
						},
					},
				},
			}),
			wantConfig: `["quay.io"]`,
		},
		{
			name: "Gov Cloud, function should add appropriate registries",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					ACRDomain:     "arointsvc.azurecr.us",
					AZEnvironment: "AzureUSGovernmentCloud",
					OperatorFlags: arov1alpha1.OperatorFlags{
						ENABLED: strconv.FormatBool(true),
					},
					Location: "eastus",
				},
			}),
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantConfig: `["arointsvc.azurecr.us","arosvc.eastus.data.azurecr.us","quay.io"]`,
		},
		{
			name: "AZEnvironment is Unset",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						ENABLED: strconv.FormatBool(true),
					},
					Location: "eastus",
				},
			}),
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantConfig: `["quay.io"]`,
		},
		{
			name:   "Both AllowedRegistries and BlockedRegistries are present, function should fail silently and not requeue",
			arocli: arocli,
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
				Spec: configv1.ImageSpec{
					RegistrySources: configv1.RegistrySources{
						BlockedRegistries: []string{
							"arointsvc.azurecr.io", "arosvc.eastus.data.azurecr.io",
						},
						AllowedRegistries: []string{
							"quay.io",
						},
					},
				},
			}),
			wantErr: `both AllowedRegistries and BlockedRegistries are present`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := &Reconciler{
				log:       log,
				arocli:    tt.arocli,
				configcli: tt.configcli,
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
				t.Error("----------------------")
				t.Error(tt.wantErr)
			}
			imgcfg, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			imgcfgjson := getRegistrySources(imgcfg)

			if string(imgcfgjson) != strings.TrimSpace(tt.wantConfig) && tt.wantConfig != "" {
				t.Error(string(imgcfgjson))
				t.Error("----------------------")
				t.Error(tt.wantConfig)
			}

		})
	}
}

func getRegistrySources(imgcfg *configv1.Image) []byte {
	var registrySourceJSON []byte
	if imgcfg.Spec.RegistrySources.AllowedRegistries != nil {
		imgRegistries := imgcfg.Spec.RegistrySources.AllowedRegistries
		sort.Strings(imgRegistries)
		registrySourceJSON, _ = json.Marshal(imgRegistries)
	} else {
		imgRegistries := imgcfg.Spec.RegistrySources.BlockedRegistries
		sort.Strings(imgRegistries)
		registrySourceJSON, _ = json.Marshal(imgRegistries)
	}
	return registrySourceJSON
}
