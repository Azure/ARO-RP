package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

func (i *Installer) ensureGenevaLogging(ctx context.Context) error {
	gl := genevalogging.NewForRP(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli)

	key, cert := i.env.ClustersGenevaLoggingSecret()

	gcsKeyBytes, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return err
	}

	gcsCertBytes, err := tls.CertAsBytes(cert)
	if err != nil {
		return err
	}

	err = gl.ApplySecret(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "certificates",
			Namespace: genevalogging.KubeNamespace,
		},
		StringData: map[string]string{
			"gcscert.pem": string(gcsCertBytes),
			"gcskey.pem":  string(gcsKeyBytes),
		},
	})
	if err != nil {
		return err
	}

	return gl.CreateOrUpdate(ctx)
}
