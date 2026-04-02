package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"sigs.k8s.io/yaml"
)

// MakeExternalKubeconfig takes an internal kubeconfig (api-int.*) and converts
// it to use the external API endpoint (api.*) with insecure-skip-tls-verify.
// This is needed because the Hive AKS cluster cannot resolve api-int.* DNS
// names (Azure Private DNS is only linked to the cluster's VNet).
func MakeExternalKubeconfig(internalKubeconfig []byte) ([]byte, error) {
	var cfg clientcmdv1.Config
	err := yaml.Unmarshal(internalKubeconfig, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal kubeconfig: %w", err)
	}

	for i := range cfg.Clusters {
		cfg.Clusters[i].Cluster.Server = strings.Replace(
			cfg.Clusters[i].Cluster.Server,
			"https://api-int.", "https://api.", 1,
		)
		// The self-signed CA does not cover the external endpoint's cert,
		// so skip TLS verification. The client cert is still used for
		// authentication (mTLS for identity, not for server verification).
		cfg.Clusters[i].Cluster.InsecureSkipTLSVerify = true
		cfg.Clusters[i].Cluster.CertificateAuthorityData = nil
	}

	return yaml.Marshal(cfg)
}
