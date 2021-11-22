package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestGenerateAROServiceKubeconfig(t *testing.T) {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	b := x509.MarshalPKCS1PrivateKey(validCaKey)

	ca := &tls.AdminKubeConfigSignerCertKey{
		SelfSignedCertKey: tls.SelfSignedCertKey{
			CertKey: tls.CertKey{
				CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
				KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
			},
		},
	}

	apiserverURL := "https://api-int.hash.rg.mydomain:6443"
	clusterName := "api-hash-rg-mydomain:6443"
	serviceName := "system:aro-service"

	adminInternalClient := &kubeconfig.AdminInternalClient{}
	adminInternalClient.Config = &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmdv1.Cluster{
					Server:                   apiserverURL,
					CertificateAuthorityData: []byte("internal API Cert"),
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: serviceName,
				Context: clientcmdv1.Context{
					Cluster:  clusterName,
					AuthInfo: serviceName,
				},
			},
		},
		CurrentContext: serviceName,
	}

	pg := graph.PersistedGraph{}

	err = pg.Set(ca, adminInternalClient)
	if err != nil {
		t.Fatal(err)
	}

	m := &manager{}

	aroServiceInternalClient, err := m.generateAROServiceKubeconfig(pg)
	if err != nil {
		t.Fatal(err)
	}

	var got *clientcmdv1.Config
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
	got.AuthInfos = []clientcmdv1.NamedAuthInfo{}
	want := adminInternalClient.Config

	if !reflect.DeepEqual(got, want) {
		t.Fatal(cmp.Diff(got, want))
	}
}

func TestGenerateUserAdminKubeconfig(t *testing.T) {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	b := x509.MarshalPKCS1PrivateKey(validCaKey)

	ca := &tls.AdminKubeConfigSignerCertKey{
		SelfSignedCertKey: tls.SelfSignedCertKey{
			CertKey: tls.CertKey{
				CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
				KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
			},
		},
	}

	apiserverURL := "https://api-int.hash.rg.mydomain:6443"
	clusterName := "api-hash-rg-mydomain:6443"
	serviceName := "system:admin"

	adminInternalClient := &kubeconfig.AdminInternalClient{}
	adminInternalClient.Config = &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmdv1.Cluster{
					Server:                   apiserverURL,
					CertificateAuthorityData: []byte("internal API Cert"),
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: serviceName,
				Context: clientcmdv1.Context{
					Cluster:  clusterName,
					AuthInfo: serviceName,
				},
			},
		},
		CurrentContext: serviceName,
	}

	pg := graph.PersistedGraph{}

	err = pg.Set(ca, adminInternalClient)
	if err != nil {
		t.Fatal(err)
	}

	m := &manager{}

	aroServiceInternalClient, err := m.generateUserAdminKubeconfig(pg)
	if err != nil {
		t.Fatal(err)
	}

	var got *clientcmdv1.Config
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
	if subject != "CN=system:admin" {
		t.Error(subject)
	}

	if !innercert[0].NotAfter.After(time.Now().AddDate(0, 11, 0)) {
		t.Error(innercert[0].NotAfter)
	}

	keyUsage := innercert[0].KeyUsage
	expectedKeyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if keyUsage != expectedKeyUsage {
		t.Error("Invalid keyUsage.")
	}

	// validate the rest of the struct
	got.AuthInfos = []clientcmdv1.NamedAuthInfo{}
	want := adminInternalClient.Config
	want.Clusters[0].Cluster.CertificateAuthorityData = nil
	want.Clusters[0].Cluster.Server = "https://api.hash.rg.mydomain:6443"

	if !reflect.DeepEqual(got, want) {
		t.Fatal(cmp.Diff(got, want))
	}
}

func TestCheckUserAdminKubeconfigUpdated(t *testing.T) {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	b := x509.MarshalPKCS1PrivateKey(validCaKey)

	ca := &tls.AdminKubeConfigSignerCertKey{
		SelfSignedCertKey: tls.SelfSignedCertKey{
			CertKey: tls.CertKey{
				CertRaw: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validCaCerts[0].Raw}),
				KeyRaw:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}),
			},
		},
	}

	for _, tt := range []struct {
		name           string
		validity       time.Duration
		mutateConfig   func(*clientcmdv1.Config) *clientcmdv1.Config
		expectedResult bool
	}{
		{
			name:           "valid and updated",
			validity:       tls.ValidityOneYear,
			expectedResult: true,
		},
		{
			name:           "clientauth cert expires soon",
			validity:       89 * tls.ValidityOneDay,
			expectedResult: false,
		},
		{
			name:     "url needs updating and cacert populated",
			validity: tls.ValidityOneYear,
			mutateConfig: func(c *clientcmdv1.Config) *clientcmdv1.Config {
				c.Clusters[0].Cluster.Server = "https://api-int.hash.rg.mydomain:6443"
				c.Clusters[0].Cluster.CertificateAuthorityData = []byte("unexpected content")
				return c
			},
			expectedResult: false,
		},
		{
			name:     "empty client key",
			validity: tls.ValidityOneYear,
			mutateConfig: func(c *clientcmdv1.Config) *clientcmdv1.Config {
				c.AuthInfos[0].AuthInfo.ClientKeyData = nil
				return c
			},
			expectedResult: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clusterName := "api-hash-rg-mydomain:6443"
			serviceName := "system:admin"

			cfg := &tls.CertCfg{
				Subject:      pkix.Name{CommonName: serviceName, Organization: nil},
				KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
				ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				Validity:     tt.validity,
			}

			var clientCertKey tls.AdminKubeConfigClientCertKey

			err = clientCertKey.SignedCertKey.Generate(cfg, ca, strings.ReplaceAll(serviceName, ":", "-"), tls.DoNotAppendParent)
			if err != nil {
				t.Fatal(err)
			}

			userAdminKubeconfig := &clientcmdv1.Config{
				Clusters: []clientcmdv1.NamedCluster{
					{
						Name: clusterName,
						Cluster: clientcmdv1.Cluster{
							Server: "https://api.hash.rg.mydomain:6443",
						},
					},
				},
				AuthInfos: []clientcmdv1.NamedAuthInfo{
					{
						Name: serviceName,
						AuthInfo: clientcmdv1.AuthInfo{
							ClientCertificateData: clientCertKey.CertRaw,
							ClientKeyData:         clientCertKey.KeyRaw,
						},
					},
				},
				Contexts: []clientcmdv1.NamedContext{
					{
						Name: serviceName,
						Context: clientcmdv1.Context{
							Cluster:  clusterName,
							AuthInfo: serviceName,
						},
					},
				},
				CurrentContext: serviceName,
			}

			if tt.mutateConfig != nil {
				userAdminKubeconfig = tt.mutateConfig(userAdminKubeconfig)
			}

			data, err := yaml.Marshal(userAdminKubeconfig)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{}

			m.doc = &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						UserAdminKubeconfig: data,
					},
				},
			}

			got := m.checkUserAdminKubeconfigUpdated()
			if tt.expectedResult != got {
				t.Errorf("Expected %t, got %t", tt.expectedResult, got)
			}
		})
	}
}
