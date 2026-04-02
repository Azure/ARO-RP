package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/require"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"sigs.k8s.io/yaml"
)

func TestMakeExternalKubeconfig(t *testing.T) {
	internalConfig := &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "test-cluster",
				Cluster: clientcmdv1.Cluster{
					Server:                   "https://api-int.test.example.com:6443",
					CertificateAuthorityData: []byte("some-ca-data"),
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: "system:aro-diagnostics",
				AuthInfo: clientcmdv1.AuthInfo{
					ClientCertificateData: []byte("cert-data"),
					ClientKeyData:         []byte("key-data"),
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: "system:aro-diagnostics",
				Context: clientcmdv1.Context{
					Cluster:  "test-cluster",
					AuthInfo: "system:aro-diagnostics",
				},
			},
		},
		CurrentContext: "system:aro-diagnostics",
	}

	internalKubeconfig, err := yaml.Marshal(internalConfig)
	require.NoError(t, err)

	externalKubeconfig, err := MakeExternalKubeconfig(internalKubeconfig)
	require.NoError(t, err)

	var got clientcmdv1.Config
	err = yaml.Unmarshal(externalKubeconfig, &got)
	require.NoError(t, err)

	// Server should be rewritten from api-int.* to api.*
	require.Equal(t, "https://api.test.example.com:6443", got.Clusters[0].Cluster.Server)

	// CA data should be stripped
	require.Nil(t, got.Clusters[0].Cluster.CertificateAuthorityData)

	// InsecureSkipTLSVerify should be set
	require.True(t, got.Clusters[0].Cluster.InsecureSkipTLSVerify)

	// Client credentials should be preserved
	require.Equal(t, []byte("cert-data"), got.AuthInfos[0].AuthInfo.ClientCertificateData)
	require.Equal(t, []byte("key-data"), got.AuthInfos[0].AuthInfo.ClientKeyData)
}

func TestMakeExternalKubeconfigNoRewriteNeeded(t *testing.T) {
	// If the server already uses api.* (not api-int.*), it should not be changed
	config := &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "test-cluster",
				Cluster: clientcmdv1.Cluster{
					Server:                   "https://api.test.example.com:6443",
					CertificateAuthorityData: []byte("some-ca-data"),
				},
			},
		},
	}

	kubeconfig, err := yaml.Marshal(config)
	require.NoError(t, err)

	result, err := MakeExternalKubeconfig(kubeconfig)
	require.NoError(t, err)

	var got clientcmdv1.Config
	err = yaml.Unmarshal(result, &got)
	require.NoError(t, err)

	// Server should remain unchanged
	require.Equal(t, "https://api.test.example.com:6443", got.Clusters[0].Cluster.Server)
}
