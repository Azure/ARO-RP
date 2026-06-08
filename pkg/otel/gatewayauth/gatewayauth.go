package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/collector/client"
	"google.golang.org/grpc/credentials"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/gateway"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/changefeed"
)

var ErrAuthenticationFailed = errors.New("authentication failed")

// gatewayAuthInfo contains the LinkID and ClusterResourceID (plus parsed
// components) that the successfully authenticated client is associated with. It
// is set both as the gRPC Peer's auth at connection time and then as the
// client's Auth in the ServerAuthenticator interface.
type gatewayAuthInfo struct {
	LinkID                string
	ClusterResourceID     string
	ClusterSubscriptionID string
	ClusterResourceGroup  string
	ClusterResourceName   string
}

func (a gatewayAuthInfo) GetAttribute(name string) any {
	switch name {
	case "clusterResourceID":
		return a.ClusterResourceID
	case "clusterSubscriptionID":
		return a.ClusterSubscriptionID
	case "clusterResourceGroup":
		return a.ClusterResourceGroup
	case "clusterResourceName":
		return a.ClusterResourceName
	default:
		return nil
	}
}

func (a gatewayAuthInfo) AuthType() string {
	return "aro-pls-linkid"
}

func (a gatewayAuthInfo) GetAttributeNames() []string {
	return []string{
		"clusterResourceID", "clusterSubscriptionID", "clusterResourceGroup", "clusterResourceName",
	}
}

var _ client.AuthData = gatewayAuthInfo{}

type authManager struct {
	_env env.Core
	log  *logrus.Entry

	// Underlying TLS transport configuration (serving key/cert)
	tls credentials.TransportCredentials

	gatewayCache              *gatewayCache
	changefeedRefreshInterval time.Duration
	changefeedBatchSize       int
}

var _ credentials.TransportCredentials = &authManager{}

func newAuthManager(
	_env env.Core,
	tls credentials.TransportCredentials,
	m metrics.Emitter,
	changefeedRefreshInterval time.Duration,
	changefeedBatchSize int,
) *authManager {
	// Align this time with the deletion in pkg/cluster/delete.go:deleteGateway
	// -- set a max of 50s so we're always in the 60s time
	if changefeedRefreshInterval > time.Second*50 {
		changefeedRefreshInterval = time.Second * 50
	}

	return &authManager{
		_env: _env,
		log:  _env.LoggerForComponent("auth"),

		tls: tls,

		gatewayCache:              newGatewayCache(_env.LoggerForComponent("changefeed"), m),
		changefeedRefreshInterval: changefeedRefreshInterval,
		changefeedBatchSize:       changefeedBatchSize,
	}
}

func (m *authManager) startChangefeed(ctx context.Context, db database.Gateway) {
	go changefeed.RunChangefeed(
		ctx, m._env.LoggerForComponent("changefeed"), db.ChangeFeed(),
		m.changefeedRefreshInterval,
		m.changefeedBatchSize, m.gatewayCache, ctx.Done(),
	)
}

// Unused as we are not a client
func (e *authManager) ClientHandshake(context.Context, string, net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, errors.New("not a client auth")
}

// ServerHandshake does the authentication handshake for servers. It returns
// the authenticated connection and the corresponding auth information about
// the connection. The auth information should embed CommonAuthInfo to return additional information
// about the credentials.
//
// If the returned net.Conn is closed, it MUST close the net.Conn provided.
func (e *authManager) ServerHandshake(c net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn := proxyproto.NewConn(c, proxyproto.WithPolicy(proxyproto.REQUIRE))

	linkID, err := gateway.LinkID(conn)
	if err != nil {
		e.log.Infof("failed reading LinkID, dropping: %s", err.Error())
		conn.Close()
		return nil, nil, err
	}

	// Ensure that we have the cache populated
	e.gatewayCache.initialPopulationWaitGroup.Wait()

	clusterResourceID, found := e.gatewayCache.clusters.Load(linkID)
	if !found {
		e.log.Infof("link ID %s was not in the db", linkID)
		conn.Close()
		return nil, nil, ErrAuthenticationFailed
	}

	parsed, err := arm.ParseResourceID(clusterResourceID)
	if err != nil {
		e.log.Errorf("failed parsing resource ID: %s", err.Error())
		conn.Close()
		return nil, nil, ErrAuthenticationFailed
	}

	authInfo := gatewayAuthInfo{
		LinkID:                linkID,
		ClusterResourceID:     clusterResourceID,
		ClusterSubscriptionID: parsed.SubscriptionID,
		ClusterResourceGroup:  parsed.ResourceGroupName,
		ClusterResourceName:   parsed.Name,
	}

	// We ignore the TLS authinfo since it does not contain any information
	// that's useful to us. If the negotiated TLS version/etc becomes relevant
	// we can add those values to the GatewayAuthInfo.
	tlsConn, _, err := e.tls.ServerHandshake(conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return tlsConn, authInfo, nil
}

// Info provides the ProtocolInfo of this TransportCredentials.
func (e *authManager) Info() credentials.ProtocolInfo {
	// proxy through the TLS information
	return e.tls.Info()
}

// Clone makes a copy of this TransportCredentials.
func (e *authManager) Clone() credentials.TransportCredentials {
	return &authManager{
		_env:                      e._env,
		log:                       e.log,
		tls:                       e.tls.Clone(),
		gatewayCache:              e.gatewayCache,
		changefeedRefreshInterval: e.changefeedRefreshInterval,
		changefeedBatchSize:       e.changefeedBatchSize,
	}
}

// Unused by gRPC
func (e *authManager) OverrideServerName(string) error {
	return nil
}
