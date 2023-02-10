package clusterauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_aad "github.com/Azure/ARO-RP/pkg/util/mocks/aad"
)

type tokenRequirements struct {
	clientID     string
	clientSecret string
	tenantID     string
	aadEndpoint  string
	resource     string
	claims       string
	signMethod   string
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
			controller := gomock.NewController(t)
			aad := mock_aad.NewMockTokenClient(controller)

			clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

			_, err := NewAzRefreshableAuthorizer(tt.log, tt.azCloudEnv, clientFake, aad)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestNewRefreshableAuthorizerToken(t *testing.T) {
	ctx := context.Background()

	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name        string
		secret      *corev1.Secret
		tr          *tokenRequirements
		getTokenErr error
		wantErr     string
	}{
		{
			name:        "fail: Invalid principal credentials",
			secret:      newV1CoreSecret(azureSecretName, nameSpace),
			tr:          newTokenRequirements(),
			getTokenErr: fmt.Errorf("400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal credentials are invalid."),
			wantErr:     "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal credentials are invalid.",
		},
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
			name: "fail: invalid signing method",
			tr: &tokenRequirements{
				clientID:     "my-client",
				clientSecret: "my-secret",
				tenantID:     "my-tenant.example.com",
				aadEndpoint:  "https://login.microsoftonline.com/",
				resource:     "https://management.azure.com/",
				claims:       `{}`,
				signMethod:   "fake-signing-method",
			},
			secret:  newV1CoreSecret(azureSecretName, nameSpace),
			wantErr: "signing method (alg) is unavailable.",
		},
		{
			name:   "pass: create new bearer authorizer token",
			tr:     newTokenRequirements(),
			secret: newV1CoreSecret(azureSecretName, nameSpace),
		},
	} {
		clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.secret).Build()

		controller := gomock.NewController(t)
		aad := mock_aad.NewMockTokenClient(controller)
		if tt.tr != nil {
			token, err := createToken(tt.tr)
			if err != nil {
				t.Errorf("failed to manually create token for mock aad")
			}
			aad.EXPECT().GetToken(ctx, log, tt.tr.clientID, tt.tr.clientSecret, tt.tr.tenantID, tt.tr.aadEndpoint, tt.tr.resource).MaxTimes(1).Return(token, tt.getTokenErr)
		}

		azRefreshAuthorizer, err := NewAzRefreshableAuthorizer(log, &azureclient.PublicCloud, clientFake, aad)
		if err != nil {
			t.Errorf("failed to create azRefreshAuthorizer, %v", err)
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

// createToken manually creates an adal.ServicePrincipalToken
func createToken(tr *tokenRequirements) (*adal.ServicePrincipalToken, error) {
	if tr.signMethod == "" {
		tr.signMethod = "HS256"
	}
	claimsEnc := base64.StdEncoding.EncodeToString([]byte(tr.claims))
	headerEnc := base64.StdEncoding.EncodeToString([]byte(`{ "alg": "` + tr.signMethod + `", "typ": "JWT" }`))
	signatureEnc := base64.StdEncoding.EncodeToString(
		hmac.New(sha512.New, []byte(headerEnc+claimsEnc+tr.clientSecret)).Sum(nil),
	)

	tk := adal.Token{}

	r := rand.New(rand.NewSource(time.Now().UnixMicro()))
	tk = adal.Token{
		AccessToken:  headerEnc + "." + claimsEnc + "." + signatureEnc,
		RefreshToken: fmt.Sprintf("rand-%d", r.Int()),
		ExpiresIn:    json.Number("300"),
		Resource:     tr.resource,
		Type:         "refresh",
	}

	aadUrl, err := url.Parse(tr.aadEndpoint)
	if err != nil {
		return nil, err
	}
	authUrl, err := url.Parse("https://login.microsoftonline.com/my-tenant.example.com/oauth2/authorize")
	if err != nil {
		return nil, err
	}
	tokenUrl, err := url.Parse("https://login.microsoftonline.com/my-tenant.example.com/oauth2/token")
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	deviceCodeUrl, err := url.Parse("https://devicecode.com")
	if err != nil {
		return nil, err
	}
	return adal.NewServicePrincipalTokenFromManualToken(adal.OAuthConfig{
		AuthorityEndpoint:  *aadUrl,
		AuthorizeEndpoint:  *authUrl,
		TokenEndpoint:      *tokenUrl,
		DeviceCodeEndpoint: *deviceCodeUrl,
	}, tr.clientID, tr.resource, tk)
}

func newTokenRequirements() *tokenRequirements {
	return &tokenRequirements{
		clientID:     "my-client",
		clientSecret: "my-secret",
		tenantID:     "my-tenant.example.com",
		aadEndpoint:  "https://login.microsoftonline.com/",
		resource:     "https://management.azure.com/",
		claims:       `{}`,
		signMethod:   "HS256",
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
