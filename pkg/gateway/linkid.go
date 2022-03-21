package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pires/go-proxyproto"
)

const (
	pp2TypeAzure                         proxyproto.PP2Type = 0xEE
	pp2SubtypeAzurePrivateEndpointLinkID byte               = 1
)

// isAllowed reads the private endpoint link ID from the haproxy binary protocol
// header injected on the front of the TCP stream by PLS.  It uses this to do a
// lookup of the gateway collection record in the in-memory cache (this is
// populated by the Cosmos DB change feed).  It then makes a decision about
// whether to allow the connection based on a static allow list and the
// additional hostnames in the gateway record. It returns the cluster ID and
// deny/allow decision.
func (g *gateway) isAllowed(conn *proxyproto.Conn, host string) (string, bool, error) {
	linkID, err := linkID(conn)
	if err != nil {
		return "", false, err
	}

	g.mu.RLock()
	gateway := g.gateways[linkID]
	g.mu.RUnlock()

	if gateway == nil || gateway.Deleting {
		return "", false, fmt.Errorf("gateway record not found for linkID %s", linkID)
	}

	// Emit a gauge for the linkID if the host is empty
	if host == "" {
		g.m.EmitGauge("gateway.nohost", 1, map[string]string{
			"linkid": linkID,
			"action": "denied",
		})
	}

	if _, found := g.allowList[strings.ToLower(host)]; found {
		return gateway.ID, true, nil
	}

	return gateway.ID,
		strings.EqualFold(host, gateway.ImageRegistryStorageAccountName+".blob."+g.env.Environment().StorageEndpointSuffix) ||
			strings.EqualFold(host, "cluster"+gateway.StorageSuffix+".blob."+g.env.Environment().StorageEndpointSuffix),
		nil
}

// linkID retrieves the private endpoint link ID from the haproxy binary
// protocol header injected on the front of the TCP stream by PLS.  See
// https://docs.microsoft.com/en-us/azure/private-link/private-link-service-overview#getting-connection-information-using-tcp-proxy-v2
func linkID(conn *proxyproto.Conn) (string, error) {
	h := conn.ProxyHeader()
	if h == nil {
		return "", errors.New("nil ProxyHeader")
	}

	tlvs, err := h.TLVs()
	if err != nil {
		return "", err
	}

	for _, tlv := range tlvs {
		if tlv.Type != pp2TypeAzure ||
			len(tlv.Value) != 5 ||
			tlv.Value[0] != pp2SubtypeAzurePrivateEndpointLinkID {
			continue
		}

		return strconv.FormatUint(uint64(binary.LittleEndian.Uint32(tlv.Value[1:])), 10), nil
	}

	return "", errors.New("link id not found")
}
