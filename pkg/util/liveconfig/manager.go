package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

type Manager interface {
	HiveRestConfig(context.Context, int) (*rest.Config, error)
}

type dev struct{}

func NewDev() Manager {
	return &dev{}
}

type prod struct {
	kv keyvault.Manager
}

func NewProd(kv keyvault.Manager) Manager {
	return &prod{
		kv: kv,
	}
}
