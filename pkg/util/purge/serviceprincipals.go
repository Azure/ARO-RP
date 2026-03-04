package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"

	msgraph_apps "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/applications"
	"github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
)

const (
	defaultKeepTag = "persist"
)

// Pattern to match V{BUILDID} which identifies e2e test runs
var buildIDPattern = regexp.MustCompile(`V\d{9,}`)

// CleanOrphanedE2EServicePrincipals removes orphaned service principals
// created during e2e test runs. It processes two types of identities:
//   - Cluster service principals (aro-v4-e2e-*)
//   - Mock MSI service principals for MIWI tests (mock-msi-*)
//
// Safety mechanisms prevent deletion of:
//   - Service principals without V{BUILDID} pattern
//   - Service principals whose resource groups have the 'persist' tag
//   - Service principals younger than the TTL
//
// This function only processes the first page of results (~100 items per prefix)
// from Microsoft Graph API. Since the cleanup runs on a schedule, orphaned resources
// will eventually be cleaned across multiple runs.
func (rc *ResourceCleaner) CleanOrphanedE2EServicePrincipals(ctx context.Context, ttl time.Duration) error {
	rc.log.Info("Starting orphaned service principal cleanup")

	rc.log.Info("Cleaning cluster service principals (prefix: aro-v4-e2e-)")
	if err := rc.cleanServicePrincipals(ctx, "aro-v4-e2e-", ttl); err != nil {
		rc.log.Errorf("Error cleaning cluster service principals: %v", err)
	}

	rc.log.Info("Cleaning mock MSI service principals (prefix: mock-msi-)")
	if err := rc.cleanServicePrincipals(ctx, "mock-msi-", ttl); err != nil {
		rc.log.Errorf("Error cleaning mock MSI service principals: %v", err)
	}

	return nil
}

func (rc *ResourceCleaner) getApplicationsByPrefix(ctx context.Context, prefix string) ([]models.Applicationable, error) {
	filter := fmt.Sprintf("startswith(displayName, '%s')", prefix)
	requestConfig := &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id", "appId", "displayName", "createdDateTime"},
		},
	}

	result, err := rc.graphClient.Applications().Get(ctx, requestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	return result.GetValue(), nil
}

func (rc *ResourceCleaner) shouldDeleteServicePrincipal(ctx context.Context, app models.Applicationable, ttl time.Duration) bool {
	displayName := *app.GetDisplayName()
	createdDateTime := app.GetCreatedDateTime()

	if createdDateTime == nil || time.Since(*createdDateTime) < ttl {
		rc.log.Infof("SKIP '%s': No createdDateTime or age < TTL", displayName)
		return false
	}

	isE2eClusterServicePrincipal := strings.HasPrefix(displayName, "aro-v4-e2e-")

	if isE2eClusterServicePrincipal && !buildIDPattern.MatchString(displayName) {
		rc.log.Infof("SKIP '%s': No V{BUILDID} pattern", displayName)
		return false
	}

	if isE2eClusterServicePrincipal {
		resourceGroupName := strings.TrimPrefix(displayName, "aro-")

		shouldKeep, reason := rc.checkSPNeededBasedOnRGStatus(ctx, resourceGroupName, ttl)
		if shouldKeep {
			rc.log.Infof("SKIP '%s': %s", displayName, reason)
			return false
		}
	}

	return true
}

func (rc *ResourceCleaner) cleanServicePrincipals(ctx context.Context, prefix string, ttl time.Duration) error {
	apps, err := rc.getApplicationsByPrefix(ctx, prefix)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		rc.log.Debugf("No applications found with prefix '%s'", prefix)
		return nil
	}

	rc.log.Infof("Found %d applications with prefix '%s'", len(apps), prefix)

	for _, app := range apps {
		if app.GetDisplayName() == nil || app.GetAppId() == nil || app.GetId() == nil {
			rc.log.Warnf("SKIP: Application missing required fields")
			continue
		}

		if !rc.shouldDeleteServicePrincipal(ctx, app, ttl) {
			continue
		}

		displayName := *app.GetDisplayName()
		appID := *app.GetAppId()
		objectID := *app.GetId()

		if rc.dryRun {
			rc.log.Infof("DRY-RUN: Would delete '%s' (appId: %s)", displayName, appID)
		} else {
			rc.log.Infof("DELETING '%s' (appId: %s)", displayName, appID)

			err = rc.graphClient.Applications().ByApplicationId(objectID).Delete(ctx, nil)
			if err != nil {
				rc.log.Errorf("ERROR deleting '%s': %v", displayName, err)
				continue
			}
			rc.log.Infof("Deleted '%s'", displayName)
		}
	}

	return nil
}

func (rc *ResourceCleaner) checkSPNeededBasedOnRGStatus(ctx context.Context, resourceGroupName string, ttl time.Duration) (bool, string) {
	group, err := rc.resourcegroupscli.Get(ctx, resourceGroupName)
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok {
			if detailedErr.StatusCode == http.StatusNotFound {
				return false, fmt.Sprintf("Resource group '%s' does not exist", resourceGroupName)
			}
		}
		rc.log.Warnf("Error checking resource group '%s': %v", resourceGroupName, err)
		return true, fmt.Sprintf("Error checking resource group '%s': %v", resourceGroupName, err)
	}

	if group.Tags != nil {
		for tagKey := range group.Tags {
			if strings.ToLower(tagKey) == defaultKeepTag {
				return true, fmt.Sprintf("Resource group '%s' has 'persist' tag", resourceGroupName)
			}
		}

		if createdAtStr, ok := group.Tags["createdAt"]; ok && createdAtStr != nil {
			createdAt, err := time.Parse(time.RFC3339Nano, *createdAtStr)
			if err != nil {
				rc.log.Warnf("Resource group '%s' has invalid createdAt tag: %v", resourceGroupName, err)
			} else {
				rgAge := time.Since(createdAt)
				if rgAge < ttl {
					return true, fmt.Sprintf("Resource group '%s' age %v < TTL %v", resourceGroupName, rgAge.Round(time.Hour), ttl)
				}
				return false, fmt.Sprintf("Resource group '%s' exists but age %v >= TTL", resourceGroupName, rgAge.Round(time.Hour))
			}
		}
	}
	return true, fmt.Sprintf("Resource group '%s' exists but has no createdAt tag", resourceGroupName)
}
