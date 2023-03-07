package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestSecurity(t *testing.T) {
	ctx := context.Background()

	validclientkey, validclientcerts, err := utiltls.GenerateKeyAndCertificate("validclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	validadminclientkey, validadminclientcerts, err := utiltls.GenerateKeyAndCertificate("validclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	l := listener.NewListener()
	defer l.Close()

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	keyvault := mock_keyvault.NewMockManager(controller)
	keyvault.EXPECT().GetCertificateSecret(gomock.Any(), env.RPServerSecretName).AnyTimes().Return(serverkey, servercerts, nil)

	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
	_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	_env.EXPECT().Hostname().AnyTimes().Return("testhost")
	_env.EXPECT().Location().AnyTimes().Return("eastus")
	_env.EXPECT().ServiceKeyvault().AnyTimes().Return(keyvault)
	_env.EXPECT().ArmClientAuthorizer().AnyTimes().Return(clientauthorizer.NewOne(validclientcerts[0].Raw))
	_env.EXPECT().AdminClientAuthorizer().AnyTimes().Return(clientauthorizer.NewOne(validadminclientcerts[0].Raw))
	_env.EXPECT().Listen().AnyTimes().Return(l, nil)
	_env.EXPECT().FeatureIsSet(env.FeatureDisableReadinessDelay).AnyTimes().Return(false)
	_env.EXPECT().FeatureIsSet(env.FeatureEnableOCMEndpoints).AnyTimes().Return(true)

	invalidclientkey, invalidclientcerts, err := utiltls.GenerateKeyAndCertificate("invalidclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	log := logrus.NewEntry(logrus.StandardLogger())
	auditHook, auditEntry := testlog.NewAudit()
	f, err := NewFrontend(ctx, auditEntry, log, _env, nil, nil, nil, nil, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// enable /healthz to return 200
	f.startTime = time.Time{}
	f.lastChangefeed.Store(time.Time{})

	go f.Run(ctx, nil, nil)

	for _, tt := range []struct {
		name              string
		url               string
		key               *rsa.PrivateKey
		cert              *x509.Certificate
		wantStatusCode    int
		wantAuditPayloads []*audit.Payload
	}{
		{
			name:           "empty url, no client certificate",
			url:            "https://server/",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, no client certificate",
			url:            "https://server/unknown",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, no client certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "admin operations url, no client certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "ready url, no client certificate",
			url:            "https://server/healthz/ready",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty url, invalid certificate",
			url:            "https://server/",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, invalid certificate",
			url:            "https://server/unknown",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, invalid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "admin operations url, invalid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "ready url, invalid certificate",
			url:            "https://server/healthz/ready",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty url, valid certificate",
			url:            "https://server/",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "empty url, valid admin certificate",
			url:            "https://server/",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, valid certificate",
			url:            "https://server/unknown",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "unknown url, valid admin certificate",
			url:            "https://server/unknown",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, valid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Success",
						ResultDescription: "Status code: 200",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "operations url, valid admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "admin operations url, valid admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusOK,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Success",
						ResultDescription: "Status code: 200",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "admin operations url, valid non-admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusForbidden,
			wantAuditPayloads: []*audit.Payload{
				{
					EnvVer:               audit.IFXAuditVersion,
					EnvName:              audit.IFXAuditName,
					EnvFlags:             257,
					EnvAppID:             audit.SourceRP,
					EnvCloudName:         _env.Environment().Name,
					EnvCloudRole:         audit.CloudRoleRP,
					EnvCloudRoleInstance: _env.Hostname(),
					EnvCloudEnvironment:  _env.Environment().Name,
					EnvCloudLocation:     _env.Location(),
					EnvCloudVer:          audit.IFXAuditCloudVer,
					CallerIdentities: []audit.CallerIdentity{
						{
							CallerDisplayName:   "",
							CallerIdentityType:  "ApplicationID",
							CallerIdentityValue: "Go-http-client/1.1",
							CallerIPAddress:     "bufferedpipe",
						},
					},
					Category:      "ResourceManagement",
					OperationName: "GET /providers/microsoft.redhatopenshift/operations",
					Result: audit.Result{
						ResultType:        "Fail",
						ResultDescription: "Status code: 403",
					},
					TargetResources: []audit.TargetResource{
						{
							TargetResourceType: "",
							TargetResourceName: "/providers/microsoft.redhatopenshift/operations",
						},
					},
				},
			},
		},
		{
			name:           "ready url, valid certificate",
			url:            "https://server/healthz/ready",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "ready url, valid admin certificate",
			url:            "https://server/healthz/ready",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer auditHook.Reset()

			tlsConfig := &tls.Config{
				RootCAs: pool,
			}
			if tt.cert != nil && tt.key != nil {
				tlsConfig.Certificates = []tls.Certificate{
					{
						Certificate: [][]byte{
							tt.cert.Raw,
						},
						PrivateKey: tt.key,
					},
				}
			}

			c := &http.Client{
				Transport: &http.Transport{
					DialContext:     l.DialContext,
					TLSClientConfig: tlsConfig,
				},
			}

			resp, err := c.Get(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			testlog.AssertAuditPayloads(t, auditHook, tt.wantAuditPayloads)
		})
	}
}
