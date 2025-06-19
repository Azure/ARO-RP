package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
		log        *logrus.Entry
		wantErr    string
	}{
		{
			name:    "fail: nil azure cloud environment",
			secret:  newV1CoreSecret(azureSecretName, nameSpace),
			wantErr: "azureEnvironment cannot be nil",
			log:     logrus.NewEntry(logrus.StandardLogger()),
		},
		{
			name:       "fail: nil log entry",
			azCloudEnv: &azureclient.PublicCloud,
			secret:     newV1CoreSecret(azureSecretName, nameSpace),
			wantErr:    "log entry cannot be nil",
		},
		{
			name:       "pass: create new azrefreshable authorizer",
			azCloudEnv: &azureclient.PublicCloud,
			secret:     newV1CoreSecret(azureSecretName, nameSpace),
			log:        logrus.NewEntry(logrus.StandardLogger()),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

			_, err := NewAzRefreshableAuthorizer(tt.log, tt.azCloudEnv, clientFake)
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
