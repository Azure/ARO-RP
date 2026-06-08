package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestServerAuthentication(t *testing.T) {
	testCases := []struct {
		desc          string
		peer          *peer.Peer
		expectedAuth  gatewayAuthInfo
		expectedError error
	}{
		{
			desc:          "no peer",
			expectedError: ErrUnableToGetPeer,
		},
		{
			desc:          "no auth info",
			peer:          &peer.Peer{},
			expectedError: ErrUnableToGetAuthInfo,
		},
		{
			desc:          "not a gateway auth",
			peer:          &peer.Peer{AuthInfo: credentials.TLSInfo{}},
			expectedError: ErrNotNegotiatedByUs,
		},
		{
			desc: "peer gatewayauthinfo is passed through into the client auth",
			peer: &peer.Peer{AuthInfo: gatewayAuthInfo{
				ClusterResourceID: "/test/resourceID",
				LinkID:            "test",
			}},
			expectedAuth: gatewayAuthInfo{
				ClusterResourceID: "/test/resourceID",
				LinkID:            "test",
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)

			e := &gatewayAuthenticationExtension{serverAuthLog: log}
			ctx := t.Context()

			if tt.peer != nil {
				ctx = peer.NewContext(ctx, tt.peer)
			}

			outCtx, err := e.Authenticate(ctx, map[string][]string{})

			if tt.expectedError != nil {
				r.ErrorIs(err, tt.expectedError)
			} else {
				r.NoError(err)

				clientInfo := client.FromContext(outCtx)

				auth, ok := clientInfo.Auth.(gatewayAuthInfo)
				r.True(ok, "gateway auth info wasn't persisted through test")

				r.Equal(tt.expectedAuth, auth)
			}
		})
	}
}
