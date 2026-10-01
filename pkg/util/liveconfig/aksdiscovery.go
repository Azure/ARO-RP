package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/api"
)

type tokenCredentialRoundTripper struct {
	next       http.RoundTripper
	credential azcore.TokenCredential
	scope      string
}

func (t *tokenCredentialRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	token, err := t.credential.GetToken(request.Context(), policy.TokenRequestOptions{
		Scopes: []string{t.scope},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS API token: %w", err)
	}

	clonedRequest := request.Clone(request.Context())
	clonedRequest.Header = request.Header.Clone()
	clonedRequest.Header.Set("Authorization", "Bearer "+token.Token)

	return t.next.RoundTrip(clonedRequest)
}

func (t *tokenCredentialRoundTripper) WrappedRoundTripper() http.RoundTripper {
	return t.next
}

// InstallerRestConfig returns the Kubernetes RestConfig for the installer execution cluster
// based on the selected backend and environment.
//
// For backend AKSJob:
//   - Production/INT: Returns RestConfig for the regional SVC AKS cluster
//   - CI: Returns RestConfig for the existing Hive-host AKS cluster (retained for CI)
//   - Dev: Can use either based on configuration
//
// For backend Hive:
//   - Returns the Hive AKS cluster RestConfig (same as HiveRestConfig)
//
// For backend Podman:
//   - Returns an error (Podman does not use Kubernetes)
func (d *dev) InstallerRestConfig(ctx context.Context) (*rest.Config, error) {
	backend, err := d.InstallerBackend(ctx)
	if err != nil {
		return nil, err
	}

	switch backend {
	case api.InstallerBackendPodman:
		return nil, fmt.Errorf("podman backend does not use Kubernetes")

	case api.InstallerBackendHive:
		// For Hive backend, use the Hive AKS cluster
		return d.HiveRestConfig(ctx, 0)

	case api.InstallerBackendAKSJob:
		// For dev, check if there's an override kubeconfig for installer execution
		installerKubeconfigPath := os.Getenv("INSTALLER_KUBE_CONFIG_PATH")
		if installerKubeconfigPath != "" {
			restConfig, err := clientcmd.BuildConfigFromFlags("", installerKubeconfigPath)
			if err != nil {
				return nil, err
			}
			return restConfig, nil
		}

		// Default to using the Hive AKS cluster for development
		// (CI will also use this - the retained Hive-host AKS cluster without Hive)
		return d.HiveRestConfig(ctx, 0)

	default:
		return nil, fmt.Errorf("unknown installer backend: %s", backend)
	}
}

func (p *prod) InstallerRestConfig(ctx context.Context) (*rest.Config, error) {
	backend, err := p.InstallerBackend(ctx)
	if err != nil {
		return nil, err
	}

	switch backend {
	case api.InstallerBackendPodman:
		return nil, fmt.Errorf("podman backend is not supported in production")

	case api.InstallerBackendHive:
		// For Hive backend, use the Hive AKS cluster
		return p.HiveRestConfig(ctx, 0)

	case api.InstallerBackendAKSJob:
		return p.installerAKSRestConfig(ctx)

	default:
		return nil, fmt.Errorf("unknown installer backend: %s", backend)
	}
}

func (p *prod) installerAKSRestConfig(ctx context.Context) (*rest.Config, error) {
	clusterName := os.Getenv(installerAKSClusterEnvVar)
	if clusterName == "" {
		return nil, fmt.Errorf("%s must be set for AKS Job installation", installerAKSClusterEnvVar)
	}

	resourceGroup := os.Getenv(installerAKSRGEnvVar)
	if resourceGroup == "" {
		return nil, fmt.Errorf("%s must be set for AKS Job installation", installerAKSRGEnvVar)
	}

	if p.tokenCredential == nil {
		return nil, fmt.Errorf("token credential is required for AKS Job installation")
	}

	p.installerCredentialsMutex.RLock()
	cached := p.cachedInstallerConfig
	cachedCluster := p.cachedInstallerCluster
	cachedResourceGroup := p.cachedInstallerRG
	p.installerCredentialsMutex.RUnlock()
	if cached != nil && cachedCluster == clusterName && cachedResourceGroup == resourceGroup {
		return rest.CopyConfig(cached), nil
	}

	p.installerCredentialsMutex.Lock()
	defer p.installerCredentialsMutex.Unlock()

	if p.cachedInstallerConfig != nil &&
		p.cachedInstallerCluster == clusterName &&
		p.cachedInstallerRG == resourceGroup {
		return rest.CopyConfig(p.cachedInstallerConfig), nil
	}

	result, err := p.managedClustersClient.ListClusterUserCredentials(ctx, resourceGroup, clusterName, "public")
	if err != nil {
		return nil, fmt.Errorf("failed to get user credentials for installer AKS cluster %s/%s: %w", resourceGroup, clusterName, err)
	}

	restConfig, err := parseKubeconfig(result.Kubeconfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user credentials for installer AKS cluster %s/%s: %w", resourceGroup, clusterName, err)
	}

	serverScope, err := aksServerScope(restConfig)
	if err != nil {
		return nil, fmt.Errorf("installer AKS cluster %s/%s must use Microsoft Entra authentication: %w", resourceGroup, clusterName, err)
	}

	restConfig.Username = ""
	restConfig.Password = ""
	restConfig.BearerToken = ""
	restConfig.BearerTokenFile = ""
	restConfig.AuthProvider = nil
	restConfig.ExecProvider = nil
	restConfig.TLSClientConfig.CertData = nil
	restConfig.TLSClientConfig.KeyData = nil
	restConfig.TLSClientConfig.CertFile = ""
	restConfig.TLSClientConfig.KeyFile = ""
	restConfig.Wrap(func(next http.RoundTripper) http.RoundTripper {
		return &tokenCredentialRoundTripper{
			next:       next,
			credential: p.tokenCredential,
			scope:      serverScope,
		}
	})

	p.cachedInstallerConfig = restConfig
	p.cachedInstallerCluster = clusterName
	p.cachedInstallerRG = resourceGroup

	return rest.CopyConfig(restConfig), nil
}

func aksServerScope(config *rest.Config) (string, error) {
	if config.ExecProvider == nil {
		return "", fmt.Errorf("user kubeconfig does not contain an exec credential provider")
	}

	for i, arg := range config.ExecProvider.Args {
		var serverID string
		switch {
		case arg == "--server-id" && i+1 < len(config.ExecProvider.Args):
			serverID = config.ExecProvider.Args[i+1]
		case strings.HasPrefix(arg, "--server-id="):
			serverID = strings.TrimPrefix(arg, "--server-id=")
		}

		if serverID != "" {
			return strings.TrimSuffix(serverID, "/") + "/.default", nil
		}
	}

	return "", fmt.Errorf("exec credential provider does not specify --server-id")
}
