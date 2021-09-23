package image

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
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
			Features: arov1alpha1.FeaturesSpec{
				ReconcileImageConfig: true,
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
	}

	for _, tt := range []*test{
		{
			name:   "Image config registry source is empty, no action",
			arocli: arocli,
			configcli: configfake.NewSimpleClientset(&configv1.Image{
				ObjectMeta: imageConfigMetadata,
			}),
			wantConfig: `{"metadata":{"name":"cluster","creationTimestamp":null},"spec":{"additionalTrustedCA":{"name":""},"registrySources":{}},"status":{}}`,
		},
		{
			name:   "allowedRegistries exists, function should add images",
			arocli: arocli,
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
			wantConfig: `{"metadata":{"name":"cluster","creationTimestamp":null},"spec":{"additionalTrustedCA":{"name":""},"registrySources":{"allowedRegistries":["quay.io","arointsvc.azurecr.io","arosvc.eastus.data.azurecr.io"]}},"status":{}}`,
		},
		{
			name:   "blockedRegistries exists, function should delete images",
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
			wantConfig: `{"metadata":{"name":"cluster","creationTimestamp":null},"spec":{"additionalTrustedCA":{"name":""},"registrySources":{"blockedRegistries":["quay.io"]}},"status":{}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := &Reconciler{
				arocli:     arocli,
				log:        log,
				jsonHandle: new(codec.JsonHandle),
				configcli:  tt.configcli,
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			imgcfg, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			imgcfgjson, _ := json.Marshal(imgcfg)

			if string(imgcfgjson) != strings.TrimSpace(tt.wantConfig) {
				t.Error(string(imgcfgjson))
				t.Error("----------------------")
				t.Error(tt.wantConfig)
			}

		})
	}
}
