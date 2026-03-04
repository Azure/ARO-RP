package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

func TLSSecretsFromKeyVault(ctx context.Context, kv azsecrets.Client, targets []types.NamespacedName, certificateName string) ([]runtime.Object, error) {
	bundle, err := kv.GetSecret(ctx, certificateName, "", nil)
	if err != nil {
		return nil, err
	}

	key, certs, err := utilpem.Parse([]byte(*bundle.Value))
	if err != nil {
		return nil, err
	}

	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	var cb []byte
	for _, cert := range certs {
		cb = append(cb, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}

	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})

	secrets := []runtime.Object{}
	for _, target := range targets {
		secrets = append(secrets, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      target.Name,
				Namespace: target.Namespace,
			},
			Data: map[string][]byte{
				corev1.TLSCertKey:       cb,
				corev1.TLSPrivateKeyKey: privateKey,
			},
			Type: corev1.SecretTypeTLS,
		})
	}

	return secrets, nil
}

func EnsureTLSSecretFromKeyvault(ctx context.Context, kv azsecrets.Client, ch clienthelper.Writer, target types.NamespacedName, certificateName string) error {
	secret, err := TLSSecretsFromKeyVault(ctx, kv, []types.NamespacedName{target}, certificateName)
	if err != nil {
		return err
	}

	return ch.Ensure(ctx, secret...)
}
