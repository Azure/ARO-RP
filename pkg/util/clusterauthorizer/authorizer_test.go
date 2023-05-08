package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type tokenRequirements struct {
	clientSecret  string
	claims        jwt.MapClaims
	signingMethod jwt.SigningMethod
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

func TestNewRefreshableAuthorizerToken(t *testing.T) {
	ctx := context.Background()

	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		secret  *corev1.Secret
		tr      *tokenRequirements
		wantErr string
	}{
		{
			name: "fail: Missing client secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_id": []byte("client-id"),
					"azure_tenant_id": []byte("tenant-id.example.com"),
				},
			},
			wantErr: "azure_client_secret does not exist in the secret",
		},
		{
			name: "pass: create new bearer authorizer token",
			tr: &tokenRequirements{
				clientSecret:  "my-secret",
				signingMethod: jwt.SigningMethodHS256,
			},
			secret: newV1CoreSecret(azureSecretName, nameSpace),
		},
	} {
		clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

		azRefreshAuthorizer, err := NewAzRefreshableAuthorizer(log, &azureclient.PublicCloud, clientFake)
		if err != nil {
			t.Errorf("failed to create azRefreshAuthorizer, %v", err)
		}
		azRefreshAuthorizer.getTokenCredential = func(*azureclient.AROEnvironment, *Credentials) (azcore.TokenCredential, error) {
			return tt.tr, nil
		}

		t.Run(tt.name, func(t *testing.T) {
			token, err := azRefreshAuthorizer.NewRefreshableAuthorizerToken(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Logf("Token: %v", token)
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestAzCredentials(t *testing.T) {
	ctx := context.Background()

	var (
		azureSecretName = "azure-credentials"
		nameSpace       = "kube-system"
	)
	for _, tt := range []struct {
		name    string
		secret  *corev1.Secret
		wantErr string
	}{
		{
			name: "fail: Missing clientID",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_secret": []byte("client-secret"),
					"azure_tenant_id":     []byte("tenant-id.example.com"),
				},
			},
			wantErr: "azure_client_id does not exist in the secret",
		},
		{
			name: "fail: missing tenantID",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_secret": []byte("client-secret"),
					"azure_client_id":     []byte("client-id"),
				},
			},
			wantErr: "azure_tenant_id does not exist in the secret",
		},
		{
			name: "fail: missing secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_id": []byte("client-id"),
					"azure_tenant_id": []byte("tenant-id.example.com"),
				},
			},
			wantErr: "azure_client_secret does not exist in the secret",
		},
		{
			name: "fail: wrong namespace",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"azure_client_secret": []byte("client-secret"),
					"azure_client_id":     []byte("client-id"),
					"azure_tenant_id":     []byte("tenant-id.example.com"),
				},
			},
			wantErr: "secrets \"azure-credentials\" not found",
		},
		{
			name: "pass: all credential properties",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      azureSecretName,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_secret": []byte("client-secret"),
					"azure_client_id":     []byte("client-id"),
					"azure_tenant_id":     []byte("tenant-id.example.com"),
				},
			},
		},
	} {
		clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

		t.Run(tt.name, func(t *testing.T) {
			_, err := AzCredentials(ctx, clientFake)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

// GetToken allows tokenRequirements to be used as an azcore.TokenCredential.
func (tr *tokenRequirements) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	token, err := jwt.NewWithClaims(tr.signingMethod, tr.claims).SignedString([]byte(tr.clientSecret))
	if err != nil {
		return azcore.AccessToken{}, err
	}

	return azcore.AccessToken{
		Token:     token,
		ExpiresOn: time.Now().Add(10 * time.Minute),
	}, nil
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
