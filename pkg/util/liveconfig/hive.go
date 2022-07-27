package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	HiveKeyvaultSecret = "HiveConfig"
)

type hiveShard struct {
	Kubeconfig []byte `json:"kubeconfig,omitempty"`
}

type hiveConfig struct {
	Shards []hiveShard `json:"shards,omitempty"`
}

func (p *prod) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	secret, err := p.kv.GetSecret(ctx, HiveKeyvaultSecret)
	if err != nil {
		return nil, err
	}

	hc := &hiveConfig{}
	err = json.Unmarshal([]byte(*secret.Value), hc)
	if err != nil {
		return nil, err
	}

	if index >= len(hc.Shards) {
		return nil, fmt.Errorf("shard '%d' does not exist", index)
	}

	clientconfig, err := clientcmd.NewClientConfigFromBytes(hc.Shards[index].Kubeconfig)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}
