package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	fake "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6/fake"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const installerUserKubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dGVzdA==
    server: https://installer.example.test:443
  name: installer
contexts:
- context:
    cluster: installer
    user: installer
  name: installer
current-context: installer
kind: Config
users:
- name: installer
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: kubelogin
      args:
      - get-token
      - --server-id
      - 6dae42f8-4368-4678-94ff-3960e28e3630
`

type staticTokenCredential struct {
	t      *testing.T
	called int
	scope  string
}

func (c *staticTokenCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.t.Helper()
	c.called++
	if len(options.Scopes) != 1 || options.Scopes[0] != c.scope {
		c.t.Fatalf("unexpected AKS token scopes: %v", options.Scopes)
	}
	return azcore.AccessToken{
		Token:     "entra-token",
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestProdInstallerRestConfig(t *testing.T) {
	t.Setenv(installerBackendEnvVar, "aks-job")
	t.Setenv(installerAKSClusterEnvVar, "aro-classic-int-eastus-svc")
	t.Setenv(installerAKSRGEnvVar, "aro-classic-int-bl-svc")

	var credentialRequests int
	server := fake.ManagedClustersServer{
		ListClusterUserCredentials: func(_ context.Context, resourceGroupName, resourceName string, options *armcontainerservice.ManagedClustersClientListClusterUserCredentialsOptions) (resp azfake.Responder[armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse], errResp azfake.ErrorResponder) {
			credentialRequests++
			if resourceGroupName != "aro-classic-int-bl-svc" {
				t.Fatalf("unexpected resource group: %s", resourceGroupName)
			}
			if resourceName != "aro-classic-int-eastus-svc" {
				t.Fatalf("unexpected cluster name: %s", resourceName)
			}
			if options == nil || options.ServerFqdn == nil || *options.ServerFqdn != "public" {
				t.Fatalf("unexpected server FQDN options: %#v", options)
			}

			response := azfake.Responder[armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse]{}
			response.SetResponse(http.StatusOK, armcontainerservice.ManagedClustersClientListClusterUserCredentialsResponse{
				CredentialResults: armcontainerservice.CredentialResults{
					Kubeconfigs: []*armcontainerservice.CredentialResult{
						{
							Name:  pointerutils.ToPtr("clusterUser"),
							Value: []byte(installerUserKubeconfig),
						},
					},
				},
			}, nil)
			return response, azfake.ErrorResponder{}
		},
	}

	managedClustersClient, err := utilcontainerservice.NewManagedClustersClientWithTransport(
		&azureclient.PublicCloud,
		"subscription",
		&azfake.TokenCredential{},
		fake.NewManagedClustersServerTransport(&server),
	)
	if err != nil {
		t.Fatal(err)
	}

	tokenCredential := &staticTokenCredential{
		t:     t,
		scope: "6dae42f8-4368-4678-94ff-3960e28e3630/.default",
	}
	manager := NewProd("eastus", managedClustersClient, tokenCredential)

	restConfig, err := manager.InstallerRestConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if restConfig.Host != "https://installer.example.test:443" {
		t.Fatalf("unexpected host: %s", restConfig.Host)
	}
	if restConfig.BearerToken != "" || restConfig.ExecProvider != nil || restConfig.AuthProvider != nil {
		t.Fatal("kubeconfig authentication was not removed")
	}
	if len(restConfig.TLSClientConfig.CertData) != 0 || len(restConfig.TLSClientConfig.KeyData) != 0 {
		t.Fatal("kubeconfig client certificate authentication was not removed")
	}

	transport := restConfig.WrapTransport(roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != "Bearer entra-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       http.NoBody,
		}, nil
	}))
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, restConfig.Host, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if tokenCredential.called != 1 {
		t.Fatalf("expected one token request, got %d", tokenCredential.called)
	}

	if _, err := manager.InstallerRestConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if credentialRequests != 1 {
		t.Fatalf("expected user credentials to be cached, got %d requests", credentialRequests)
	}
}

func TestProdInstallerRestConfigRequiresConfiguration(t *testing.T) {
	t.Setenv(installerBackendEnvVar, "aks-job")
	t.Setenv(installerAKSClusterEnvVar, "")
	t.Setenv(installerAKSRGEnvVar, "")

	manager := NewProd("eastus", nil, &staticTokenCredential{t: t})
	if _, err := manager.InstallerRestConfig(context.Background()); err == nil {
		t.Fatal("expected missing installer AKS configuration to fail")
	}
}

func TestAKSServerScope(t *testing.T) {
	for _, tt := range []struct {
		name    string
		config  *rest.Config
		want    string
		wantErr bool
	}{
		{
			name: "separate argument",
			config: &rest.Config{
				ExecProvider: &clientcmdapi.ExecConfig{
					Args: []string{"get-token", "--server-id", "server-id"},
				},
			},
			want: "server-id/.default",
		},
		{
			name: "equals argument",
			config: &rest.Config{
				ExecProvider: &clientcmdapi.ExecConfig{
					Args: []string{"get-token", "--server-id=server-id/"},
				},
			},
			want: "server-id/.default",
		},
		{
			name:    "no exec provider",
			config:  &rest.Config{},
			wantErr: true,
		},
		{
			name: "no server ID",
			config: &rest.Config{
				ExecProvider: &clientcmdapi.ExecConfig{Args: []string{"get-token"}},
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aksServerScope(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("aksServerScope() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("aksServerScope() = %q, want %q", got, tt.want)
			}
		})
	}
}
