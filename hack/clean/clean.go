package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

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
	"v4-eastus-aks1",
	"v4-australiasoutheast-aks1",
	"v4-westeurope-aks1",
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
	err := env.ValidateVars(
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID")
	if err != nil {
		return err
	}

	env, err := env.NewCoreForCI(ctx, log, env.SERVICE_TOOLING)
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

	createdTag := defaultCreatedAtTag
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

	err = rc.CleanOrphanedE2EServicePrincipals(ctx, settings.ttl)
	if err != nil {
		log.Errorf("Error cleaning orphaned service principals: %v", err)
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

func normalizeTagsCaseInsensitive(tags map[string]*string) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(tags))

	for k, v := range tags {
		key := strings.ToLower(k)
		if v == nil {
			normalized[key] = ""
			continue
		}

		normalized[key] = *v
	}

	return normalized
}

func isTruthyTagValue(value string) bool {
	normalized := strings.TrimSpace(value)

	truthy, err := strconv.ParseBool(normalized)
	if err == nil {
		return truthy
	}

	return strings.EqualFold(normalized, "true")
}

func (s settings) shouldDelete(resourceGroup mgmtfeatures.ResourceGroup, log *logrus.Entry) bool {
	// don't mess with clusters in RGs managed by a production RP. Although
	// the production deny assignment will prevent us from breaking most
	// things, that does not include us potentially detaching the cluster's
	// NSG from the vnet, thus breaking inbound access to the cluster.
	// Historically we only evaluated resource groups that had a "purge" tag
	// (dev clusters). That gate was removed so prod e2e clusters are also
	// considered for deletion by the subsequent TTL/persist/createdAt checks.
	if resourceGroup.Name == nil || *resourceGroup.Name == "" {
		log.Warnf("Group with empty name cannot be evaluated. SKIP.")
		return false
	}
	name := *resourceGroup.Name

	// if prefix is set we check if we need to evaluate this group for purge
	// before we check other fields.
	if len(s.deleteGroupPrefixes) > 0 {
		isDeleteGroup := false
		for _, deleteGroupPrefix := range s.deleteGroupPrefixes {
			if strings.HasPrefix(name, deleteGroupPrefix) {
				isDeleteGroup = true
				break
			}
		}
		// return if prefix not matched
		if !isDeleteGroup {
			return false
		}
	}

	normalizedTags := normalizeTagsCaseInsensitive(resourceGroup.Tags)

	keepTagValue, keepTagExists := normalizedTags[strings.ToLower(defaultKeepTag)]
	if keepTagExists && isTruthyTagValue(keepTagValue) {
		log.Infof("Group %s is to persist. SKIP.", name)
		return false
	}

	createdAtValue, ok := normalizedTags[strings.ToLower(s.createdTag)]
	if !ok || createdAtValue == "" {
		log.Infof("Group %s does not have %s tag. SKIP.", name, s.createdTag)
		return false
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtValue)
	if err != nil {
		log.Infof("%s: %s", name, err)
		return false
	}
	if time.Since(createdAt) < s.ttl {
		log.Infof("Group %s is still less than TTL. SKIP.", name)
		return false
	}

	if contains(denylist, name) {
		return false
	}

	return true
}
