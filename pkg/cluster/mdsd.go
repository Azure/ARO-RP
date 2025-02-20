package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

// "Rotate" the MDSD certificates in the cluster. The copy that is rotated from
// comes from RP-Config, and is only able to be updated when the RP is updated.
func RenewMDSDCertificate(ctx context.Context, log *logrus.Entry, _env env.Interface, ch clienthelper.Interface) error {
	key, cert := _env.ClusterGenevaLoggingSecret()
	gcsKeyBytes, err := utilpem.Encode(key)
	if err != nil {
		return err
	}
	gcsCertBytes, err := utilpem.Encode(cert)
	if err != nil {
		return err
	}

	s := &corev1.Secret{}
	err = ch.GetOne(
		ctx, types.NamespacedName{Name: pkgoperator.SecretName, Namespace: pkgoperator.Namespace}, s,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch operator secret object: %w", err)
	}

	s.Data["gcscert.pem"] = gcsCertBytes
	s.Data["gcskey.pem"] = gcsKeyBytes

	err = ch.Ensure(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to update MDSD certificate: %w", err)
	}

	return nil
}
