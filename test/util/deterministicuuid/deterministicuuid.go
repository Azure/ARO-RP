package deterministicuuid

import (
	"bytes"

	gofrsuuid "github.com/gofrs/uuid"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	_ uint8 = iota
	CLUSTERS
	ASYNCOPERATIONS
	PORTAL
	GATEWAY
	OPENSHIFT_VERSIONS
	CLUSTERMANAGER
	MAINTENANCE_MANIFESTS
)

type gen struct {
	namespace uint8
	counter   uint16
}

// NewTestUUIDGenerator returns a uuid.Generator which generates UUIDv4s
// suitable for testing.
func NewTestUUIDGenerator(namespace uint8) uuid.Generator {
	return &gen{
		namespace: namespace,
	}
}

// Generate generates a UUID that increments each call, using a counter to
// specify the last two bytes and namespaced by the first byte.
func (g *gen) Generate() string {
	g.counter++

	// repeat the namespace for the first 14 bytes to make an obvious non-random
	// pattern
	uuidBytes := bytes.Repeat([]byte{g.namespace}, 14)

	// 16 bits of uuid ought to be enough for any test :)
	uuidBytes = append(uuidBytes, byte(g.counter>>8))
	uuidBytes = append(uuidBytes, byte(g.counter))
	return gofrsuuid.FromBytesOrNil(uuidBytes).String()
}
