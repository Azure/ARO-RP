package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"reflect"
	"testing"

	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

func TestGenerateAROServiceKubeconfig(t *testing.T) {
	ctx := context.Background()
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	b := x509.MarshalPKCS1PrivateKey(validCaKey)

	g := graph{
		reflect.TypeOf(&tls.AdminKubeConfigSignerCertKey{}): &tls.AdminKubeConfigSignerCertKey{
			tls.SelfSignedCertKey{
				tls.CertKey{
					CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
					KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
				},
			},
		},
		reflect.TypeOf(&kubeconfig.AdminInternalClient{}): &kubeconfig.AdminInternalClient{},
	}

	APIServerUrl := "https://api.hash.rg.mydomain:6443"
	clusterName := "api-hash-rg-mydomain:6443"
	serviceName := "system:aro-service"

	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	adminInternalClient.Config = &clientcmd.Config{
		Clusters: []clientcmd.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmd.Cluster{
					Server:                   APIServerUrl,
					CertificateAuthorityData: nil,
				},
			},
		},
		AuthInfos: []clientcmd.NamedAuthInfo{},
		Contexts: []clientcmd.NamedContext{
			{
				Name: serviceName,
				Context: clientcmd.Context{
					Cluster:  clusterName,
					AuthInfo: serviceName,
				},
			},
		},
		CurrentContext: serviceName,
	}

	i := &Installer{}

	aroServiceInternalClient, err := i.generateAROServiceKubeconfig(ctx, g, serviceName)
	if err != nil {
		t.Fatal(err)
	}

	var got *clientcmd.Config
	err = yaml.Unmarshal(aroServiceInternalClient.File.Data, &got)
	if err != nil {
		t.Fatal(err)
	}

	innerpem := string(got.AuthInfos[0].AuthInfo.ClientCertificateData) + string(got.AuthInfos[0].AuthInfo.ClientKeyData)
	innerkey, innercert, err := utilpem.Parse([]byte(innerpem))
	if err != nil {
		t.Fatal(err)
	}

	// validate the result in 2 stages: first verify the key and certificate
	// are valid (signed by CA, have proper validity period, etc)
	// then remove the AuthInfo struct from the result and validate
	// rest of the fields by comparing with the template.

	err = innercert[0].CheckSignatureFrom(validCaCerts[0])
	if err != nil {
		t.Error("Failed to parse certificate", err)
	}

	issuer := innercert[0].Issuer.String()
	expectedIssuer := "CN=validca"
	if issuer != expectedIssuer {
		t.Errorf("Invalid certificate issuer, want: %s, got %s", expectedIssuer, issuer)
	}

	subject := innercert[0].Subject.String()
	expectedSubject := "CN=system:aro-service,O=system:masters"
	if subject != expectedSubject {
		t.Errorf("Invalid subject want: %s, got %s", expectedSubject, subject)
	}

	// TODO - more data to validate notBefore, notAfter and add validation for the key
	_ = innerkey

	got.AuthInfos = make([]clientcmd.NamedAuthInfo, 0, 0)
	want := adminInternalClient.Config

	if !reflect.DeepEqual(got, want) {
		t.Fatal("invalid internal client")
	}
}
