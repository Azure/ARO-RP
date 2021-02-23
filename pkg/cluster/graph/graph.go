package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	"github.com/openshift/installer/pkg/asset"
)

// Graph is used to generate and persist the graph as a one-off.  For subsequent
// uses, use PersistedGraph.

type Graph map[string]asset.Asset

func (g Graph) Get(a asset.Asset) asset.Asset {
	return g[reflect.TypeOf(a).String()]
}

func (g Graph) Set(as ...asset.Asset) {
	for _, a := range as {
		g[reflect.TypeOf(a).String()] = a
	}
}

func (g Graph) Resolve(a asset.Asset) error {
	if g.Get(a) != nil {
		return nil
	}

	for _, dep := range a.Dependencies() {
		err := g.Resolve(dep)
		if err != nil {
			return err
		}
	}

	parents := asset.Parents{}
	for _, v := range g {
		parents[reflect.TypeOf(v)] = v
	}

	err := a.Generate(parents)
	if err != nil {
		return err
	}

	g.Set(a)

	return nil
}
