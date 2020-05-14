package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/purge"
)

var (
	dryRun = flag.Bool("dryRun", true, `Dry run`)
)

const (
	defaultTTL          = 48 * time.Hour
	defaultCreatedAtTag = "createdAt"
	defaultKeepTag      = "persist"
)

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	flag.Parse()

	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, log *logrus.Entry) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	var ttl time.Duration
	if os.Getenv("AZURE_PURGE_TTL") != "" {
		var err error
		ttl, err = time.ParseDuration(os.Getenv("AZURE_PURGE_TTL"))
		if err != nil {
			return err
		}
	} else {
		ttl = defaultTTL
	}

	var createdTag = defaultCreatedAtTag
	if os.Getenv("AZURE_PURGE_CREATED_TAG") != "" {
		createdTag = os.Getenv("AZURE_PURGE_CREATED_TAG")
	}

	purgeEnvironment := os.Getenv("AZURE_PURGE_ENVIRONMENT")

	shouldDelete := func(resourceGroup mgmtfeatures.ResourceGroup, log *logrus.Entry) bool {
		if purgeEnvironment == "e2e" {
			// in e2e environment, we only want to work on marked groups
			if _, ok := resourceGroup.Tags["aroe2e"]; !ok {
				return false
			}
		}

		for t := range resourceGroup.Tags {
			if strings.ToLower(t) == defaultKeepTag {
				log.Debugf("Group %s is to persist. SKIP.", *resourceGroup.Name)
				return false
			}
		}

		// azure tags is not consistent with lower/upper cases.
		if _, ok := resourceGroup.Tags[createdTag]; !ok {
			log.Debugf("Group %s does not have createdAt tag. SKIP.", *resourceGroup.Name)
			return false
		}

		createdAt, err := time.Parse(time.RFC3339Nano, *resourceGroup.Tags[createdTag])
		if err != nil {
			log.Errorf("%s: %s", *resourceGroup.Name, err)
			return false
		}
		if time.Now().Sub(createdAt) < ttl {
			log.Debugf("Group %s is still less than TTL. SKIP.", *resourceGroup.Name)
			return false
		}

		return true
	}

	log.Infof("Starting the resource cleaner, DryRun: %t", *dryRun)

	rc, err := purge.NewResourceCleaner(log, subscriptionID, shouldDelete, *dryRun)
	if err != nil {
		return err
	}

	return rc.CleanResourceGroups(ctx)
}
