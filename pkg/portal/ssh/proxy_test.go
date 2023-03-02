package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/bufferedpipe"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// fakeClient runs a fake client on the given connection.  It validates the
// server key, authenticates, writes a ping request, reads a pong reply, then
// closes the connection
func fakeClient(c net.Conn, serverKey *rsa.PublicKey, user string, password string) error {
	publicKey, err := cryptossh.NewPublicKey(serverKey)
	if err != nil {
		return err
	}

	conn, _, _, err := cryptossh.NewClientConn(c, "", &cryptossh.ClientConfig{
		HostKeyCallback: cryptossh.FixedHostKey(publicKey),
		User:            user,
		Auth: []cryptossh.AuthMethod{
			cryptossh.Password(password),
		},
	})
	if err != nil {
		return err
	}

	_, reply, err := conn.SendRequest("ping", true, []byte("ping"))
	if err != nil {
		return err
	}

	if string(reply) != "pong" {
		return fmt.Errorf("invalid reply %q", string(reply))
	}

	return conn.Close()
}

// fakeServer returns a test listener for an SSH server which validates the
// client key, reads ping request(s) and writes pong replies
func fakeServer(clientKey *rsa.PublicKey) (*listener.Listener, error) {
	l := listener.NewListener()

	clientPublicKey, err := cryptossh.NewPublicKey(clientKey)
	if err != nil {
		return nil, err
	}

	config := &cryptossh.ServerConfig{
		PublicKeyCallback: func(conn cryptossh.ConnMetadata, key cryptossh.PublicKey) (*cryptossh.Permissions, error) {
			if conn.User() != "core" {
				return nil, fmt.Errorf("invalid user")
			}
			if !bytes.Equal(key.Marshal(), clientPublicKey.Marshal()) {
				return nil, fmt.Errorf("invalid key")
			}
			return nil, nil
		},
	}

	key, _, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		return nil, err
	}

	signer, err := cryptossh.NewSignerFromSigner(key)
	if err != nil {
		return nil, err
	}

	config.AddHostKey(signer)

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			go func() {
				conn, _, requests, err := cryptossh.NewServerConn(c, config)
				if err != nil {
					return
				}

				go func() {
					for request := range requests {
						if request.Type == "ping" && request.WantReply {
							err := request.Reply(true, []byte("pong"))
							if err != nil {
								break
							}
						} else {
							err := request.Reply(false, nil)
							if err != nil {
								break
							}
						}
					}
				}()

				_ = conn.Wait()
			}()
		}
	}()

	return l, nil
}

func TestProxy(t *testing.T) {
	ctx := context.Background()
	username := "test"
	password := "00000000-0000-0000-0000-000000000000"
	subscriptionID := "10000000-0000-0000-0000-000000000000"
	resourceGroup := "rg"
	resourceName := "cluster"
	resourceID := "/subscriptions/" + subscriptionID + "/resourcegroups/" + resourceGroup + "/providers/microsoft.redhatopenshift/openshiftclusters/" + resourceName
	apiServerPrivateEndpointIP := "1.2.3.4"

	hostKey, _, err := utiltls.GenerateKeyAndCertificate("proxy", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	clusterKey, _, err := utiltls.GenerateKeyAndCertificate("cluster", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	l, err := fakeServer(&clusterKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	goodOpenShiftClusterDocument := func() *api.OpenShiftClusterDocument {
		return &api.OpenShiftClusterDocument{
			ID:  resourceID,
			Key: resourceID,
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						APIServerPrivateEndpointIP: apiServerPrivateEndpointIP,
					},
					SSHKey: api.SecureBytes(x509.MarshalPKCS1PrivateKey(clusterKey)),
				},
			},
		}
	}

	goodPortalDocument := func(id string) *api.PortalDocument {
		return &api.PortalDocument{
			ID: id,
			Portal: &api.Portal{
				ID:       resourceID,
				Username: username,
				SSH: &api.SSH{
					Master: 1,
				},
			},
		}
	}

	type test struct {
		name           string
		username       string
		password       string
		fixtureChecker func(*test, *testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient, *cosmosdb.FakePortalDocumentClient)
		mocks          func(*mock_proxy.MockDialer)
		wantErrPrefix  string
		wantLogs       []map[string]types.GomegaMatcher
	}

	for _, tt := range []*test{
		{
			name:     "good",
			username: username,
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := goodPortalDocument(tt.password)
				fixture.AddPortalDocuments(portalDocument)
				openShiftClusterDocument := goodOpenShiftClusterDocument()
				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				portalDocument = goodPortalDocument(tt.password)
				portalDocument.Portal.SSH.Authenticated = true
				checker.AddPortalDocuments(portalDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			mocks: func(dialer *mock_proxy.MockDialer) {
				dialer.EXPECT().DialContext(gomock.Any(), "tcp", apiServerPrivateEndpointIP+":2201").Return(l.DialContext(ctx, "", ""))
			},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.InfoLevel),
					"msg":         gomega.Equal("authentication succeeded"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
				{
					"level":           gomega.Equal(logrus.InfoLevel),
					"msg":             gomega.Equal("connected"),
					"hostname":        gomega.Equal("master-1"),
					"resource_group":  gomega.Equal(resourceGroup),
					"resource_id":     gomega.Equal(resourceID),
					"resource_name":   gomega.Equal(resourceName),
					"subscription_id": gomega.Equal(subscriptionID),
					"username":        gomega.Equal(username),
				},
				{
					"level":           gomega.Equal(logrus.InfoLevel),
					"msg":             gomega.Equal("disconnected"),
					"duration":        gomega.BeNumerically(">", 0),
					"hostname":        gomega.Equal("master-1"),
					"resource_group":  gomega.Equal(resourceGroup),
					"resource_id":     gomega.Equal(resourceID),
					"resource_name":   gomega.Equal(resourceName),
					"subscription_id": gomega.Equal(subscriptionID),
					"username":        gomega.Equal(username),
				},
			},
		},
		{
			name:     "bad username",
			username: "bad",
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := goodPortalDocument(tt.password)
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
			},
			wantErrPrefix: "ssh: handshake failed",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.WarnLevel),
					"msg":         gomega.Equal("authentication failed"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal("bad"),
				},
			},
		},
		{
			name:          "bad password, not uuid",
			username:      username,
			password:      "bad",
			wantErrPrefix: "ssh: handshake failed",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.WarnLevel),
					"msg":         gomega.Equal("authentication failed"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
		{
			name:          "bad password",
			username:      username,
			password:      password,
			wantErrPrefix: "ssh: handshake failed",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.WarnLevel),
					"msg":         gomega.Equal("authentication failed"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
		{
			name:     "not ssh record",
			username: username,
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := goodPortalDocument(tt.password)
				portalDocument.Portal.SSH = nil
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
			},
			wantErrPrefix: "ssh: handshake failed",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.WarnLevel),
					"msg":         gomega.Equal("authentication failed"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
		{
			name:     "sad openshiftClusters database",
			username: username,
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := goodPortalDocument(tt.password)
				fixture.AddPortalDocuments(portalDocument)
				portalDocument = goodPortalDocument(tt.password)
				portalDocument.Portal.SSH.Authenticated = true
				checker.AddPortalDocuments(portalDocument)

				openShiftClustersClient.SetError(fmt.Errorf("sad"))
			},
			wantErrPrefix: "EOF",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.InfoLevel),
					"msg":         gomega.Equal("authentication succeeded"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
		{
			name:     "sad portal database",
			username: username,
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalClient.SetError(fmt.Errorf("sad"))
			},
			wantErrPrefix: "ssh: handshake failed",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.WarnLevel),
					"msg":         gomega.Equal("authentication failed"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
		{
			name:     "sad dialer",
			username: username,
			password: password,
			fixtureChecker: func(tt *test, fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := goodPortalDocument(tt.password)
				fixture.AddPortalDocuments(portalDocument)
				openShiftClusterDocument := goodOpenShiftClusterDocument()
				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				portalDocument = goodPortalDocument(tt.password)
				portalDocument.Portal.SSH.Authenticated = true
				checker.AddPortalDocuments(portalDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			mocks: func(dialer *mock_proxy.MockDialer) {
				dialer.EXPECT().DialContext(gomock.Any(), "tcp", apiServerPrivateEndpointIP+":2201").Return(nil, fmt.Errorf("sad"))
			},
			wantErrPrefix: "EOF",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level":       gomega.Equal(logrus.InfoLevel),
					"msg":         gomega.Equal("authentication succeeded"),
					"remote_addr": gomega.Not(gomega.BeEmpty()),
					"username":    gomega.Equal(username),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dbPortal, portalClient := testdatabase.NewFakePortal()
			dbOpenShiftClusters, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()

			fixture := testdatabase.NewFixture().
				WithOpenShiftClusters(dbOpenShiftClusters).
				WithPortal(dbPortal)

			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(tt, fixture, checker, openShiftClustersClient, portalClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			client, client1 := bufferedpipe.New()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dialer := mock_proxy.NewMockDialer(ctrl)

			if tt.mocks != nil {
				tt.mocks(dialer)
			}

			hook, log := testlog.New()

			s, err := New(nil, nil, log, nil, hostKey, nil, dbOpenShiftClusters, dbPortal, dialer)
			if err != nil {
				t.Fatal(err)
			}

			r := mux.NewRouter()
			r.Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/ssh/new").HandlerFunc(s.New)

			done := make(chan struct{})

			go func() {
				_ = s.newConn(context.Background(), client1)
				close(done)
			}()

			err = fakeClient(client, &hostKey.PublicKey, tt.username, tt.password)
			if err != nil && !strings.HasPrefix(err.Error(), tt.wantErrPrefix) ||
				err == nil && tt.wantErrPrefix != "" {
				t.Error(err)
			}

			<-done

			openShiftClustersClient.SetError(nil)
			portalClient.SetError(nil)

			for _, err = range checker.CheckOpenShiftClusters(openShiftClustersClient) {
				t.Error(err)
			}

			for _, err = range checker.CheckPortals(portalClient) {
				t.Error(err)
			}

			err = testlog.AssertLoggingOutput(hook, tt.wantLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
