package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var (
	azureSecretName = "azure-credentials"
	nameSpace       = "kube-system"
)

func TestNewAzRefreshableAuthorizer(t *testing.T) {
	for _, tt := range []struct {
		name       string
		azCloudEnv *azureclient.AROEnvironment
		secret     *corev1.Secret
		wantErr    string
	}{
		{
			name:    "fail: nil azure cloud environment",
			secret:  newV1CoreSecret(azureSecretName, nameSpace),
			wantErr: "azureEnvironment cannot be nil",
		},
		{
			name:       "pass: create new azrefreshable authorizer",
			azCloudEnv: &azureclient.PublicCloud,
			secret:     newV1CoreSecret(azureSecretName, nameSpace),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.LogForTesting(t)

			clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

			_, err := NewAzRefreshableAuthorizer(log, tt.azCloudEnv, clientFake)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func newV1CoreSecret(azSecretName, ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      azSecretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"azure_client_secret": []byte("my-secret"),
			"azure_client_id":     []byte("my-client"),
			"azure_tenant_id":     []byte("my-tenant.example.com"),
		},
	}
}
