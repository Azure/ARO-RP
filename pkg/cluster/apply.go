package cluster

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

func EnsureTLSSecretFromKeyvault(ctx context.Context, env env.Interface, ch clienthelper.Interface, target types.NamespacedName, certificateName string) error {
	bundle, err := env.ClusterKeyvault().GetSecret(ctx, certificateName)
	if err != nil {
		return err
	}

	key, certs, err := utilpem.Parse([]byte(*bundle.Value))
	if err != nil {
		return err
	}

	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	var cb []byte
	for _, cert := range certs {
		cb = append(cb, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      target.Name,
			Namespace: target.Namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       cb,
			corev1.TLSPrivateKeyKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
		},
		Type: corev1.SecretTypeTLS,
	}

	return ch.Ensure(ctx, secret)
}
