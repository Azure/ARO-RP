package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/pem"
)

type ServiceKeyvaultInterface interface {
	GetCertificateSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, []*x509.Certificate, error)
	GetSecret(ctx context.Context, secretName string) ([]byte, error)
}

type serviceKeyvault struct {
	cli basekeyvault.BaseClient
	uri string
}

func NewServiceKeyvault(ctx context.Context, im instancemetadata.InstanceMetadata) (ServiceKeyvaultInterface, error) {
	kvAuthorizer, err := RPAuthorizer(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	kv := &serviceKeyvault{
		cli: basekeyvault.New(kvAuthorizer),
	}

	kv.uri, err = GetVaultURI(ctx, im, generator.ServiceKeyVaultTagValue)
	if err != nil {
		return nil, err
	}

	return kv, nil
}

func (kv *serviceKeyvault) GetCertificateSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, []*x509.Certificate, error) {
	bundle, err := kv.cli.GetSecret(ctx, kv.uri, secretName, "")
	if err != nil {
		return nil, nil, err
	}

	key, certs, err := pem.Parse([]byte(*bundle.Value))
	if err != nil {
		return nil, nil, err
	}

	if key == nil {
		return nil, nil, fmt.Errorf("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate found")
	}

	return key, certs, nil
}

func (kv *serviceKeyvault) GetSecret(ctx context.Context, secretName string) ([]byte, error) {
	bundle, err := kv.cli.GetSecret(ctx, kv.uri, secretName, "")
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(*bundle.Value)
}

func GetVaultURI(ctx context.Context, im instancemetadata.InstanceMetadata, tag string) (string, error) {
	rpAuthorizer, err := RPAuthorizer(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	vaults := keyvault.NewVaultsClient(im.SubscriptionID(), rpAuthorizer)

	vs, err := vaults.ListByResourceGroup(ctx, im.ResourceGroup(), nil)
	if err != nil {
		return "", err
	}

	var count int
	var uri string
	for _, v := range vs {
		if v.Tags[generator.KeyVaultTagName] != nil &&
			*v.Tags[generator.KeyVaultTagName] == tag {
			uri = *v.Properties.VaultURI
			count++
		}
	}

	if count != 1 {
		return "", fmt.Errorf("found %d key vaults with vault tag value %s, expected 1", count, tag)
	}

	return uri, nil
}
