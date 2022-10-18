package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
	"github.com/sirupsen/logrus"
)

type Manager interface {
	HiveRestConfig(context.Context, int) (*rest.Config, error)
	InstallViaHive(context.Context) (bool, error)
	AdoptByHive(context.Context) (bool, error)

	// Allows overriding the default installer pullspec for Prod, if the OpenShiftVersions database is not populated
	DefaultInstallerPullSpecOverride(context.Context) string
}

type dev struct{}

func NewDev() Manager {
	return &dev{}
}

type prod struct {
	location              string
	managedClustersClient containerservice.ManagedClustersClient

	hiveCredentialsMutex *sync.RWMutex
	cachedCredentials    map[int]*rest.Config
}

func NewProd(location string, managedClustersClient containerservice.ManagedClustersClient, log *logrus.Entry) Manager {
	p := &prod{
		location:              location,
		managedClustersClient: managedClustersClient,
		cachedCredentials:     make(map[int]*rest.Config),
		hiveCredentialsMutex:  &sync.RWMutex{},
	}
	go p.hiveConfigRetrieveLoop(1, log)
	return p
}

func (p *prod) hiveConfigRetrieveLoop(index int, log *logrus.Entry) {

	successWaitTime := 360 * time.Second
	failureWaitTime := 60 * time.Second
	apiCallTimeout := 10 * time.Second

	for {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, apiCallTimeout)

		timeToSleep := successWaitTime
		if !p.hiveConfigOne(ctx, index, log) {
			timeToSleep = failureWaitTime
		}

		cancel() // can't defer
		time.Sleep(timeToSleep)

	}
}

func (p *prod) hiveConfigOne(ctx context.Context, index int, log *logrus.Entry) bool {

	rpResourceGroup := fmt.Sprintf("rp-%s", p.location)
	rpResourceName := fmt.Sprintf("aro-aks-cluster-%03d", index)

	res, err := p.managedClustersClient.ListClusterUserCredentials(ctx, rpResourceGroup, rpResourceName, "")
	if err != nil {
		return false
	}

	parsed, err := parseKubeconfig(*res.Kubeconfigs)
	if err != nil {
		log.Info(err)
		return false
	}

	p.hiveCredentialsMutex.Lock()
	p.cachedCredentials[index] = parsed
	p.hiveCredentialsMutex.Unlock()

	return true

}
