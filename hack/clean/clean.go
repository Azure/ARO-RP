package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/purge"
)

// denylist exists as belt and braces protection for important RGs, even though
// they may already have the persist=true tag set, especially if it is easy to
// accidentally redeploy the RG without the persist=true tag set.
var denylist = []string{
	"v4-eastus",
	"v4-australiasoutheast",
	"v4-westeurope",
	"management-westeurope",
	"management-eastus",
	"management-australiasoutheast",
	"images",
	"secrets",
	"dns",
}

const (
	defaultTTL          = 48 * time.Hour
	defaultCreatedAtTag = "createdAt"
	defaultKeepTag      = "persist"
)

func main() {
	dryRun := flag.Bool("dryRun", true, `Dry run`)

	flag.Parse()
	ctx := context.Background()
	log := utillog.GetLogger()

	if err := run(ctx, log, dryRun); err != nil {
		log.Fatal(err)
	}
}

type settings struct {
	ttl                 time.Duration
	createdTag          string
	deleteGroupPrefixes []string
}

func run(ctx context.Context, log *logrus.Entry, dryRun *bool) error {
	for _, key := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	env, err := env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

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

	deleteGroupPrefixes := []string{}
	if os.Getenv("AZURE_PURGE_RESOURCEGROUP_PREFIXES") != "" {
		deleteGroupPrefixes = strings.Split(os.Getenv("AZURE_PURGE_RESOURCEGROUP_PREFIXES"), ",")
	}

	settings := settings{
		deleteGroupPrefixes: deleteGroupPrefixes,
		ttl:                 ttl,
		createdTag:          createdTag,
	}

	log.Infof("Starting the resource cleaner, DryRun: %t", *dryRun)

	rc, err := purge.NewResourceCleaner(log, env, settings.shouldDelete, *dryRun)
	if err != nil {
		return err
	}

	return rc.CleanResourceGroups(ctx)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (s settings) shouldDelete(resourceGroup mgmtfeatures.ResourceGroup, log *logrus.Entry) bool {
	//assume its a prod cluster, dev clusters will have purge tag
	devCluster := false
	if resourceGroup.Tags != nil {
		_, devCluster = resourceGroup.Tags["purge"]
	}

	// don't mess with clusters in RGs managed by a production RP. Although
	// the production deny assignment will prevent us from breaking most
	// things, that does not include us potentially detaching the cluster's
	// NSG from the vnet, thus breaking inbound access to the cluster.
	if !devCluster && resourceGroup.ManagedBy != nil && *resourceGroup.ManagedBy != "" {
		return false
	}

	// if prefix is set we check if we need to evaluate this group for purge
	// before we check other fields.
	if len(s.deleteGroupPrefixes) > 0 {
		isDeleteGroup := false
		for _, deleteGroupPrefix := range s.deleteGroupPrefixes {
			if strings.HasPrefix(*resourceGroup.Name, deleteGroupPrefix) {
				isDeleteGroup = true
				break
			}
		}
		// return if prefix not matched
		if !isDeleteGroup {
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
	if _, ok := resourceGroup.Tags[s.createdTag]; !ok {
		log.Debugf("Group %s does not have createdAt tag. SKIP.", *resourceGroup.Name)
		return false
	}

	createdAt, err := time.Parse(time.RFC3339Nano, *resourceGroup.Tags[s.createdTag])
	if err != nil {
		log.Errorf("%s: %s", *resourceGroup.Name, err)
		return false
	}
	if time.Since(createdAt) < s.ttl {
		log.Debugf("Group %s is still less than TTL. SKIP.", *resourceGroup.Name)
		return false
	}

	// TODO(mj): Fix this!
	if contains(denylist, *resourceGroup.Name) {
		return false
	}

	return true
}
