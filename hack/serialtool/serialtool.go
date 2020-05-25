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
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type serialTool struct {
	log *logrus.Entry
	oc  *api.OpenShiftCluster

	accounts        storage.AccountsClient
	virtualMachines compute.VirtualMachinesClient

	clusterResourceGroup string
	infraID              string
}

func newSerialTool(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (*serialTool, error) {
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

	return &serialTool{
		log: log,
		oc:  oc,

		accounts:        storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),
		virtualMachines: compute.NewVirtualMachinesClient(r.SubscriptionID, fpAuthorizer),

		clusterResourceGroup: stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/'),
		infraID:              infraID,
	}, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s resourceid vmname\n", os.Args[0])
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

	oc, err := getCluster(ctx, log, env, os.Args[1])
	if err != nil {
		return err
	}

	s, err := newSerialTool(log, env, oc)
	if err != nil {
		return err
	}

	return s.dump(ctx, os.Args[2])
}

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}
