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
)

const (
	defaultKeepTag = "persist"
)

var (
	// Pattern to match V{BUILDID} which identifies e2e test runs
	buildIDPattern = regexp.MustCompile(`V\d{9,}`)
)

func (rc *ResourceCleaner) CleanOrphanedServicePrincipals(ctx context.Context, ttl time.Duration) error {
	rc.log.Info("Starting orphaned service principal cleanup")

	prefixes := []struct {
		prefix      string
		description string
	}{
		{"aro-v4-e2e-", "Cluster service principals"},
		{"v4-e2e-", "Disk encryption set managed identities"},
		{"mock-msi-", "Mock MSI service principals (MIWI e2e tests)"},
	}

	for _, p := range prefixes {
		rc.log.Infof("Cleaning %s (prefix: %s)", p.description, p.prefix)
		if err := rc.cleanServicePrincipalsByPrefix(ctx, p.prefix, ttl); err != nil {
			rc.log.Errorf("Error cleaning prefix '%s': %v", p.prefix, err)
		}
	}

	return nil
}

func (rc *ResourceCleaner) cleanServicePrincipalsByPrefix(ctx context.Context, prefix string, ttl time.Duration) error {
	// List all applications with the specified prefix
	filter := fmt.Sprintf("startswith(displayName, '%s')", prefix)
	requestConfig := &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id", "appId", "displayName", "createdDateTime"},
		},
	}

	result, err := rc.graphClient.Applications().Get(ctx, requestConfig)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	apps := result.GetValue()
	if len(apps) == 0 {
		rc.log.Debugf("No applications found with prefix '%s'", prefix)
		return nil
	}

	rc.log.Infof("Found %d applications with prefix '%s'", len(apps), prefix)

	for _, app := range apps {
		displayName := ""
		if app.GetDisplayName() != nil {
			displayName = *app.GetDisplayName()
		}

		appID := ""
		if app.GetAppId() != nil {
			appID = *app.GetAppId()
		}

		objectID := ""
		if app.GetId() != nil {
			objectID = *app.GetId()
		}

		// mock-msi service principals don't have build IDs or associated resource groups
		isMockMSI := strings.HasPrefix(displayName, "mock-msi-")

		// For non-mock-msi SPs, verify this is from an e2e test run by checking for V{BUILDID} pattern
		if !isMockMSI && !buildIDPattern.MatchString(displayName) {
			rc.log.Infof("SKIP '%s': No V{BUILDID} pattern - likely infrastructure", displayName)
			continue
		}

		createdDateTime := app.GetCreatedDateTime()
		if createdDateTime == nil {
			rc.log.Warnf("SKIP '%s': No createdDateTime", displayName)
			continue
		}

		age := time.Since(*createdDateTime)

		if age < ttl {
			rc.log.Debugf("SKIP '%s': Age %v < TTL %v", displayName, age.Round(time.Hour), ttl)
			continue
		}

		// mock-msi SPs don't have associated resource groups, so skip RG check for them
		if !isMockMSI {
			resourceGroupName := rc.determineResourceGroupName(displayName)

			shouldKeep, reason := rc.shouldKeepServicePrincipal(ctx, resourceGroupName, ttl)
			if shouldKeep {
				rc.log.Infof("SKIP '%s': %s", displayName, reason)
				continue
			}
		}

		if rc.dryRun {
			rc.log.Infof("DRY-RUN: Would delete '%s' (appId: %s, age: %v)", displayName, appID, age.Round(time.Hour))
		} else {
			rc.log.Infof("DELETING '%s' (appId: %s, age: %v)", displayName, appID, age.Round(time.Hour))
			err := rc.graphClient.Applications().ByApplicationId(objectID).Delete(ctx, nil)
			if err != nil {
				rc.log.Errorf("ERROR deleting '%s': %v", displayName, err)
				continue
			}
			rc.log.Infof("SUCCESS: Deleted '%s'", displayName)
		}
	}

	return nil
}

func (rc *ResourceCleaner) determineResourceGroupName(displayName string) string {
	// For service principals: aro-v4-e2e-V{BUILDID}-{LOCATION}[-miwi][-prod-csp][-prod-miwi]
	// Resource group is: v4-e2e-V{BUILDID}-{LOCATION}[-miwi][-prod-csp][-prod-miwi]
	if strings.HasPrefix(displayName, "aro-") {
		return strings.TrimPrefix(displayName, "aro-")
	}

	// For disk encryption sets: v4-e2e-V{BUILDID}-{LOCATION}[-miwi][-prod-csp][-prod-miwi]-disk-encryption-set
	// Resource group is: v4-e2e-V{BUILDID}-{LOCATION}[-miwi][-prod-csp][-prod-miwi] (without -disk-encryption-set suffix)
	if strings.HasSuffix(displayName, "-disk-encryption-set") {
		return strings.TrimSuffix(displayName, "-disk-encryption-set")
	}
	return displayName
}

// shouldKeepServicePrincipal checks if the service principal should be kept based on resource group status
func (rc *ResourceCleaner) shouldKeepServicePrincipal(ctx context.Context, resourceGroupName string, ttl time.Duration) (bool, string) {
	group, err := rc.resourcegroupscli.Get(ctx, resourceGroupName)
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok {
			if detailedErr.StatusCode == http.StatusNotFound {
				return false, "Resource group does not exist"
			}
		}
		rc.log.Warnf("Error checking resource group '%s': %v", resourceGroupName, err)
		return true, fmt.Sprintf("Error checking resource group: %v", err)
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
