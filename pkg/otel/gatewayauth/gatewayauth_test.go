package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/credentials"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/gateway"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	testmetrics "github.com/Azure/ARO-RP/test/util/metrics"
)

func TestServerHandshake(t *testing.T) {
	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	noTLVProxyV2, err := proxyproto.HeaderProxyFromAddrs(2, nil, nil).Format()
	require.NoError(t, err)

	proxyV2WithTLVStruct := proxyproto.HeaderProxyFromAddrs(2, nil, nil)
	err = proxyV2WithTLVStruct.SetTLVs([]proxyproto.TLV{
		{Type: proxyproto.PP2Type(0xEE), Value: []byte{0x01, 0x01, 0x23, 0x45, 0x67}},
	})
	require.NoError(t, err)

	proxyV2WithTLV, err := proxyV2WithTLVStruct.Format()
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	fakeLinkID := "1732584193"
	fakeSubscriptionID := "00000000-0000-0000-0000-000000000000"
	fakeResourceGroup := "resourceGroup"
	fakeResourceName := "resourceName"
	fakeResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", fakeSubscriptionID, fakeResourceGroup, fakeResourceName)

	testCases := []struct {
		desc string

		clientWrite []byte

		expectedError  error
		expectedGauges []testmetrics.MetricsAssertion[int64]
	}{
		{
			desc:           "not proxy v2 causes an error",
			clientWrite:    []byte{0xFF, 0xFF, 0xFF},
			expectedError:  gateway.ErrNilProxyHeader,
			expectedGauges: []testmetrics.MetricsAssertion[int64]{},
		},
		{
			desc:          "proxyv2 with no TLV causes an error",
			clientWrite:   noTLVProxyV2,
			expectedError: gateway.ErrLinkIDNotFound,
		},
		{
			desc:        "correct TLV and linkID succeeds",
			clientWrite: proxyV2WithTLV,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			r := require.New(t)
			ctx := t.Context()

			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_, log := testlog.LogForTesting(t)

			metrics := testmetrics.NewFakeMetricsEmitter(t)

			_env.EXPECT().LoggerForComponent(gomock.Any()).AnyTimes().DoAndReturn(func(c string) *logrus.Entry {
				return log.WithField("component", c)
			})

			serverTLS := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{
				{PrivateKey: serverkey, Certificate: [][]byte{servercerts[0].Raw}},
			}})

			m := newAuthManager(_env, serverTLS, metrics, time.Second, 100)
			// add our expected link ID

			m.gatewayCache.OnDoc(&api.GatewayDocument{ID: fakeLinkID, Gateway: &api.Gateway{ID: fakeResourceID}})
			m.gatewayCache.OnAllPendingProcessed(true)

			l := listener.NewListener()
			defer l.Close()

			go func() {
				client, err := l.DialContext(ctx, "", "")
				if err != nil {
					t.Error(err)
				}

				client.Write(tt.clientWrite)

				// If we're not erroring out, attempt to write some test data to verify we're talking TLS
				if tt.expectedError == nil {
					tlsClient := tls.Client(client, &tls.Config{RootCAs: pool, ServerName: "server", NextProtos: []string{"h2"}})

					err = tlsClient.Handshake()
					if err != nil {
						t.Error(err)
					}

					state := tlsClient.ConnectionState()
					if !state.HandshakeComplete || state.ServerName != "server" {
						t.Errorf("got %s, not 'server'", state.ServerName)
					}

					tlsClient.Write([]byte("hello!"))
					tlsClient.Close()
				}
			}()

			c, err := l.Accept()
			r.NoError(err)

			wrappedConn, authInfo, err := m.ServerHandshake(c)

			if tt.expectedError != nil {
				r.ErrorIs(err, tt.expectedError)

				// no conn or auth info should be returned
				r.Nil(wrappedConn)
				r.Nil(authInfo)

				// should be closed
				_, err = c.Write([]byte{})
				r.ErrorContains(err, "connection closed")
			} else {
				r.NoError(err)

				// auth type should be ours
				r.Equal("aro-pls-linkid", authInfo.AuthType())

				// AuthInfo should be populated with the parsed Azure Resource ID
				ourAuth, ok := authInfo.(gatewayAuthInfo)
				r.True(ok, "authInfo wasn't ours")

				r.Equal(strings.ToLower(fakeResourceID), ourAuth.ClusterResourceID)
				r.Equal(strings.ToLower(fakeSubscriptionID), ourAuth.ClusterSubscriptionID)
				r.Equal(strings.ToLower(fakeResourceName), ourAuth.ClusterResourceName)
				r.Equal(strings.ToLower(fakeResourceGroup), ourAuth.ClusterResourceGroup)

				// verify we can talk over it with the TLS client returned
				buf := new(bytes.Buffer)
				r.Eventually(func() bool {
					_, err = buf.ReadFrom(wrappedConn)
					r.NoError(err)
					return assert.Equal(t, []byte("hello!"), buf.Bytes())
				}, time.Second, time.Millisecond*100)
			}

			metrics.AssertFloats()

			g := append(tt.expectedGauges, testmetrics.MetricsAssertion[int64]{
				MetricName: "changefeed.caches.size",
				Dimensions: map[string]string{"name": "GatewayDocument"},
				Value:      1,
			})

			metrics.AssertGauges(g...)
		})
	}
}
