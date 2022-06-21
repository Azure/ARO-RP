package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"math/big"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) ensureSSHKey(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, mutateSSHKey)

	return err
}

func mutateSSHKey(doc *api.OpenShiftClusterDocument) error {
	if doc.OpenShiftCluster.Properties.SSHKey != nil {
		return nil
	}

	sshKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	doc.OpenShiftCluster.Properties.SSHKey = x509.MarshalPKCS1PrivateKey(sshKey)

	return nil
}

func randomLowerCaseAlphanumericStringWithNoVowels(n int) (string, error) {
	return randomString("bcdfghjklmnpqrstvwxyz0123456789", n)
}

func randomString(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}
