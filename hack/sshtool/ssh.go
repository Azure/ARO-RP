package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type sshTool struct {
	log *logrus.Entry
	oc  *api.OpenShiftCluster

	interfaces    network.InterfacesClient
	loadBalancers network.LoadBalancersClient

	clusterResourceGroup string
	infraID              string
}

func newSSHTool(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (*sshTool, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(oc.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	infraID := oc.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	return &sshTool{
		log: log,
		oc:  oc,

		interfaces:    network.NewInterfacesClient(r.SubscriptionID, fpAuthorizer),
		loadBalancers: network.NewLoadBalancersClient(r.SubscriptionID, fpAuthorizer),

		clusterResourceGroup: stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/'),
		infraID:              infraID,
	}, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n")
	fmt.Fprintf(os.Stderr, "  %s enable resourceid\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s agent resourceid\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s disable resourceid\n", os.Args[0])
}

func getCluster(ctx context.Context, log *logrus.Entry, _env env.Interface, resourceID string) (*api.OpenShiftCluster, error) {
	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDatabase(ctx, log.WithField("component", "database"), _env, &noop.Noop{}, cipher, uuid.NewV4().String())
	if err != nil {
		return nil, err
	}

	doc, err := db.OpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("resource %q not found", resourceID)
	}

	return doc.OpenShiftCluster, nil
}

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 3 {
		usage()
		os.Exit(2)
	}

	env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	oc, err := getCluster(ctx, log, env, os.Args[2])
	if err != nil {
		return err
	}

	s, err := newSSHTool(log, env, oc)
	if err != nil {
		return err
	}

	switch strings.ToLower(os.Args[1]) {
	case "agent":
		return s.agent(ctx)
	case "disable":
		return s.disable(ctx)
	case "enable":
		return s.enable(ctx)
	default:
		usage()
		os.Exit(2)
	}

	return nil
}

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}
