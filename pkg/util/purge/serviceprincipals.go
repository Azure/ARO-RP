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
	msgraph_models "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	msgraph_sps "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/serviceprincipals"
)

const (
	defaultKeepTag = "persist"
)

var (
	// Pattern to match V{BUILDID} which identifies e2e test runs
	buildIDPattern = regexp.MustCompile(`V\d{9,}`)
)

// CleanOrphanedE2EServicePrincipals removes orphaned service principals
// created during e2e test runs. It processes three types of identities:
//   - Cluster service principals (aro-v4-e2e-*)
//   - Disk encryption set  (v4-e2e-*-disk-encryption-set)
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
	if err := rc.cleanServicePrincipals(ctx, "aro-v4-e2e-", "", ttl); err != nil {
		rc.log.Errorf("Error cleaning cluster service principals: %v", err)
	}

	rc.log.Info("Cleaning disk encryption set managed identities (prefix: v4-e2e-, suffix: -disk-encryption-set)")
	if err := rc.cleanServicePrincipals(ctx, "v4-e2e-", "-disk-encryption-set", ttl); err != nil {
		rc.log.Errorf("Error cleaning disk encryption set identities: %v", err)
	}

	rc.log.Info("Cleaning mock MSI service principals (prefix: mock-msi-)")
	if err := rc.cleanServicePrincipals(ctx, "mock-msi-", "", ttl); err != nil {
		rc.log.Errorf("Error cleaning mock MSI service principals: %v", err)
	}

	return nil
}

func (rc *ResourceCleaner) listByDisplayName(ctx context.Context, prefix string, suffix string, isServicePrincipal bool) ([]interface{}, error) {
	var filter string
	if suffix != "" {
		filter = fmt.Sprintf("startswith(displayName, '%s') and endswith(displayName, '%s')", prefix, suffix)
	} else {
		filter = fmt.Sprintf("startswith(displayName, '%s')", prefix)
	}

	if isServicePrincipal {
		requestConfig := &msgraph_sps.ServicePrincipalsRequestBuilderGetRequestConfiguration{
			QueryParameters: &msgraph_sps.ServicePrincipalsRequestBuilderGetQueryParameters{
				Filter: &filter,
				Select: []string{"id", "appId"},
			},
		}

		result, err := rc.graphClient.ServicePrincipals().Get(ctx, requestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to list service principals: %w", err)
		}

		sps := result.GetValue()
		items := make([]interface{}, len(sps))
		for i, sp := range sps {
			items[i] = sp
		}
		return items, nil
	}

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

	apps := result.GetValue()
	items := make([]interface{}, len(apps))
	for i, app := range apps {
		items[i] = app
	}
	return items, nil
}

func (rc *ResourceCleaner) cleanServicePrincipals(ctx context.Context, prefix string, suffix string, ttl time.Duration) error {
	// Disk encryption sets are service principals, not applications
	isServicePrincipal := suffix != ""

	items, err := rc.listByDisplayName(ctx, prefix, suffix, isServicePrincipal)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		rc.log.Debugf("No items found with prefix '%s'", prefix)
		return nil
	}

	rc.log.Infof("Found %d items with prefix '%s'", len(items), prefix)

	for _, item := range items {
		var displayName, appID, objectID string
		var createdDateTime *time.Time

		if isServicePrincipal {
			sp := item.(msgraph_models.ServicePrincipalable)

			if sp.GetAppId() != nil {
				appID = *sp.GetAppId()
			}

			if sp.GetId() != nil {
				objectID = *sp.GetId()
			}

			spDetails, err := rc.graphClient.ServicePrincipals().ByServicePrincipalId(objectID).Get(ctx, nil)
			if err != nil {
				rc.log.Warnf("SKIP service principal (objectID: %s): Failed to get details: %v", objectID, err)
				continue
			}

			if val, err := spDetails.GetBackingStore().Get("displayName"); err == nil && val != nil {
				if strVal, ok := val.(*string); ok && strVal != nil {
					displayName = *strVal
				} else {
					rc.log.Debugf("Service principal (objectID: %s): displayName type assertion failed", objectID)
				}
			} else {
				rc.log.Debugf("Service principal (objectID: %s): Failed to get displayName from BackingStore: %v", objectID, err)
			}

			if val, err := spDetails.GetBackingStore().Get("createdDateTime"); err == nil && val != nil {
				if timeVal, ok := val.(*time.Time); ok {
					createdDateTime = timeVal
				} else {
					rc.log.Debugf("Service principal (objectID: %s): createdDateTime type assertion failed", objectID)
				}
			} else {
				rc.log.Debugf("Service principal (objectID: %s): Failed to get createdDateTime from BackingStore: %v", objectID, err)
			}
		} else {
			app := item.(msgraph_models.Applicationable)

			if app.GetDisplayName() != nil {
				displayName = *app.GetDisplayName()
			}

			if app.GetAppId() != nil {
				appID = *app.GetAppId()
			}

			if app.GetId() != nil {
				objectID = *app.GetId()
			}

			createdDateTime = app.GetCreatedDateTime()
		}

		isMockMSI := strings.HasPrefix(displayName, "mock-msi-")

		if !isMockMSI && !buildIDPattern.MatchString(displayName) {
			rc.log.Infof("SKIP '%s': No V{BUILDID} pattern", displayName)
			continue
		} else if createdDateTime == nil {
			rc.log.Warnf("SKIP '%s' (objectID: %s): No createdDateTime", displayName, objectID)
			continue
		}

		age := time.Since(*createdDateTime)

		if age < ttl {
			rc.log.Debugf("SKIP '%s': Age %v < TTL %v", displayName, age.Round(time.Hour), ttl)
			continue
		} else if !isMockMSI {
			resourceGroupName := determineResourceGroupName(displayName)

			shouldKeep, reason := rc.checkSPNeededBasedOnRGStatus(ctx, resourceGroupName, ttl)
			if shouldKeep {
				rc.log.Infof("SKIP '%s': %s", displayName, reason)
				continue
			}
		}

		if rc.dryRun {
			rc.log.Infof("DRY-RUN: Would delete '%s' (appId: %s, age: %v)", displayName, appID, age.Round(time.Hour))
		} else {
			rc.log.Infof("DELETING '%s' (appId: %s, age: %v)", displayName, appID, age.Round(time.Hour))

			var err error
			if isServicePrincipal {
				err = rc.graphClient.ServicePrincipals().ByServicePrincipalId(objectID).Delete(ctx, nil)
			} else {
				err = rc.graphClient.Applications().ByApplicationId(objectID).Delete(ctx, nil)
			}

			if err != nil {
				rc.log.Errorf("ERROR deleting '%s': %v", displayName, err)
				continue
			}
			rc.log.Infof("SUCCESS: Deleted '%s'", displayName)
		}
	}

	return nil
}

func determineResourceGroupName(displayName string) string {
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
