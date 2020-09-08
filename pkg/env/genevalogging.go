package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

type ClustersGenevaLoggingInterface interface {
	ClustersGenevaLoggingSecret() (key []byte, cert []byte)
}

type genevalogging struct {
	key  []byte
	cert []byte
}

func (gl *genevalogging) ClustersGenevaLoggingSecret() (key []byte, cert []byte) {
	return gl.key, gl.cert
}

func NewClustersGenevaLogging(ctx context.Context, kv keyvault.Manager) (ClustersGenevaLoggingInterface, error) {
	key, certs, err := kv.GetCertificateSecret(ctx, ClusterLoggingSecretName)
	if err != nil {
		return nil, err
	}

	keyb, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return nil, err
	}

	certb, err := tls.CertAsBytes(certs[0])
	if err != nil {
		return nil, err
	}

	return &genevalogging{
		key:  keyb,
		cert: certb,
	}, nil
}
