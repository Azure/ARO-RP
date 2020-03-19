package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"crypto/x509"
	"encoding/pem"
	"reflect"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"

	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestGenerateAROServiceKubeconfig(t *testing.T) {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	b := x509.MarshalPKCS1PrivateKey(validCaKey)

	g := graph{
		reflect.TypeOf(&tls.AdminKubeConfigSignerCertKey{}): &tls.AdminKubeConfigSignerCertKey{
			SelfSignedCertKey: tls.SelfSignedCertKey{
				CertKey: tls.CertKey{
					CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
					KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
				},
			},
		},
		reflect.TypeOf(&kubeconfig.AdminInternalClient{}): &kubeconfig.AdminInternalClient{},
	}

	apiserverURL := "https://api.hash.rg.mydomain:6443"
	clusterName := "api-hash-rg-mydomain:6443"
	serviceName := "system:aro-service"

	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	adminInternalClient.Config = &clientcmd.Config{
		Clusters: []clientcmd.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmd.Cluster{
					Server:                   apiserverURL,
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

	aroServiceInternalClient, err := i.generateAROServiceKubeconfig(g)
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
	if innerkey == nil {
		t.Error("Client Key is invalid.")
	}

	// validate the result in 2 stages: first verify the key and certificate
	// are valid (signed by CA, have proper validity period, etc)
	// then remove the AuthInfo struct from the result and validate
	// rest of the fields by comparing with the template.

	err = innercert[0].CheckSignatureFrom(validCaCerts[0])
	if err != nil {
		t.Fatal(err)
	}

	issuer := innercert[0].Issuer.String()
	if issuer != "CN=validca" {
		t.Error(issuer)
	}

	subject := innercert[0].Subject.String()
	if subject != "CN=system:aro-service,O=system:masters" {
		t.Error(subject)
	}

	if !innercert[0].NotAfter.After(time.Now().AddDate(9, 11, 0)) {
		t.Error(innercert[0].NotAfter)
	}

	keyUsage := innercert[0].KeyUsage
	expectedKeyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if keyUsage != expectedKeyUsage {
		t.Error("Invalid keyUsage.")
	}

	// validate the rest of the struct
	got.AuthInfos = []clientcmd.NamedAuthInfo{}
	want := adminInternalClient.Config

	if !reflect.DeepEqual(got, want) {
		t.Fatal("invalid internal client.")
	}
}
