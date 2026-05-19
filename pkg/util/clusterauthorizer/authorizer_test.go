package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGetTokenCredential(t *testing.T) {
	for _, tt := range []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name: "workload identity: returns WorkloadIdentityCredential when AZURE_FEDERATED_TOKEN_FILE is set",
			envVars: map[string]string{
				"AZURE_FEDERATED_TOKEN_FILE": "/var/run/secrets/openshift/serviceaccount/token",
				"AZURE_CLIENT_ID":            "test-client-id",
				"AZURE_TENANT_ID":            "test-tenant-id",
			},
		},
		{
			name: "service principal: returns EnvironmentCredential when AZURE_CLIENT_SECRET is set",
			envVars: map[string]string{
				"AZURE_CLIENT_ID":     "test-client-id",
				"AZURE_TENANT_ID":     "test-tenant-id",
				"AZURE_CLIENT_SECRET": "test-secret",
			},
		},
		{
			name:    "error: returns error when no credential env vars are set",
			envVars: map[string]string{},
			wantErr: "missing environment variable AZURE_TENANT_ID",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Unset relevant env vars and restore after test
			for _, key := range []string{
				"AZURE_FEDERATED_TOKEN_FILE",
				"AZURE_CLIENT_ID",
				"AZURE_TENANT_ID",
				"AZURE_CLIENT_SECRET",
			} {
				orig, existed := os.LookupEnv(key)
				os.Unsetenv(key)
				if existed {
					t.Cleanup(func() { os.Setenv(key, orig) })
				}
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cred, err := GetTokenCredential(&azureclient.PublicCloud)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if tt.wantErr == "" && cred == nil {
				t.Error("expected non-nil credential")
			}
		})
	}
}

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
