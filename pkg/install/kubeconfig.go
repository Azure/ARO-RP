package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

// addKubeconfigContext adds new entry to existing kubeconfig Config
func (i *Installer) addKubeconfigContext(adminInternalClient *kubeconfig.AdminInternalClient, cert []byte, key []byte, name string) error {
	aroSystemAuthInfo := clientcmd.NamedAuthInfo{
		Name: name,
		AuthInfo: clientcmd.AuthInfo{
			ClientCertificateData: cert,
			ClientKeyData:         key,
		},
	}
	aroNamedContext := clientcmd.NamedContext{
		Name: name,
		Context: clientcmd.Context{
			Cluster:  adminInternalClient.Config.Contexts[0].Context.Cluster,
			AuthInfo: name,
		},
	}

	adminInternalClient.Config.AuthInfos = append(adminInternalClient.Config.AuthInfos, aroSystemAuthInfo)
	adminInternalClient.Config.Contexts = append(adminInternalClient.Config.Contexts, aroNamedContext)

	return nil
}

// generate generates kubeconfig file from provided Config
func (i *Installer) generateKubeconfig(adminInternalClient *kubeconfig.AdminInternalClient) error {
	data, err := yaml.Marshal(adminInternalClient.Config)
	if err != nil {
		return err
	}

	adminInternalClient.File = &asset.File{
		Filename: adminInternalClient.File.Filename,
		Data:     data,
	}
	return nil
}
