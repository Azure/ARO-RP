package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type aksInstallationManager struct {
	log *logrus.Entry
	env env.Core

	kubernetescli kubernetes.Interface

	dh dynamichelper.Interface
}

// NewAKSManagerFromHiveManager creates an AKS installation manager from the
// Hive ClusterManager
func NewAKSManagerFromHiveManager(h ClusterManager) (*aksInstallationManager, error) {
	m, ok := h.(*clusterManager)
	if !ok {
		return nil, errors.New("not a Hive clustermanager?")
	}

	return &aksInstallationManager{
		log: m.log,
		env: m.env,

		kubernetescli: m.kubernetescli,

		dh: m.dh,
	}, nil
}

func (c *aksInstallationManager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error {
	sppSecret, err := servicePrincipalSecretForInstall(doc.OpenShiftCluster, sub, c.env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	psSecret, err := pullsecretSecret(doc.OpenShiftCluster.Properties.HiveProfile.Namespace, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	// todo: install

	resources := []kruntime.Object{
		sppSecret,
		psSecret,
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	err = c.dh.Ensure(ctx, resources...)
	if err != nil {
		return err
	}

	return nil
}
