package deterministicuuid

import (
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
	// 16 bits of uuid ought to be enough for any test :)
	return gofrsuuid.FromBytesOrNil([]byte{
		g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, g.namespace, byte(uint8(g.counter >> 8)), byte(uint8(g.counter)),
	}).String()
}
