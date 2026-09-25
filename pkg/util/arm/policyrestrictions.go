package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkpolicyinsights "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/policyinsights/armpolicyinsights"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armpolicyinsights"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const mismatchDetailsPrefix = "Unexpected property mutations detected, likely due to Azure policies. Details: "

var mutatingPolicyEffects = map[string]struct{}{
	"append": {},
	"modify": {},
	"mutate": {},
}

func isMutatingEffect(effect string) bool {
	_, ok := mutatingPolicyEffects[strings.ToLower(effect)]
	return ok
}

func EnrichMismatchesWithPolicyContext(
	ctx context.Context,
	log *logrus.Entry,
	policyClient armpolicyinsights.PolicyRestrictionsClient,
	resourceGroupName string,
	mismatches []Mismatch,
) {
	if policyClient == nil || len(mismatches) == 0 {
		return
	}

	type resKey struct {
		name, resourceType string
	}
	cache := make(map[resKey][]PolicyAttribution)

	for i := range mismatches {
		name := mismatches[i].ResourceName
		resourceType := mismatches[i].ResourceType
		if name == "" || resourceType == "" {
			continue
		}
		key := resKey{name: strings.ToLower(name), resourceType: strings.ToLower(resourceType)}
		policies, cached := cache[key]
		if !cached {
			apiVersion := azureclient.APIVersion(resourceType)
			if apiVersion == "" {
				log.Warnf("no checkPolicyRestrictions apiVersion registered for %s, skipping enrichment for %s", resourceType, name)
				cache[key] = nil
				continue
			}
			policies = lookupPolicyRestrictions(ctx, log, policyClient, resourceGroupName, resourceType, name, apiVersion)
			cache[key] = policies
		}
		if len(policies) > 0 {
			mismatches[i].Policies = policies
		}
	}
}

func lookupPolicyRestrictions(
	ctx context.Context,
	log *logrus.Entry,
	client armpolicyinsights.PolicyRestrictionsClient,
	resourceGroupName string,
	resourceType string,
	resourceName string,
	apiVersion string,
) []PolicyAttribution {
	content := map[string]interface{}{
		"type": resourceType,
		"name": resourceName,
	}
	details := &sdkpolicyinsights.CheckRestrictionsResourceDetails{
		ResourceContent: content,
		APIVersion:      pointerutils.ToPtr(apiVersion),
	}
	resp, err := client.CheckAtResourceGroupScope(ctx, resourceGroupName, sdkpolicyinsights.CheckRestrictionsRequest{
		ResourceDetails: details,
	}, nil)
	if err != nil {
		log.Errorf("checkPolicyRestrictions failed for %s/%s: %v", resourceType, resourceName, err)
		return nil
	}
	if resp.FieldRestrictions == nil {
		return nil
	}

	var out []PolicyAttribution
	for _, fr := range resp.FieldRestrictions {
		field := ""
		if fr.Field != nil {
			field = *fr.Field
		}
		for _, restriction := range fr.Restrictions {
			if restriction.PolicyEffect == nil {
				continue
			}
			effect := *restriction.PolicyEffect
			if !isMutatingEffect(effect) {
				continue
			}
			entry := PolicyAttribution{Field: field, Effect: effect}
			if restriction.Result != nil {
				entry.Result = string(*restriction.Result)
			}
			for _, v := range restriction.Values {
				if v != nil {
					entry.Values = append(entry.Values, *v)
				}
			}
			if restriction.Policy != nil {
				if restriction.Policy.PolicyDefinitionID != nil {
					entry.PolicyDefinition = parsePolicyRef(log, *restriction.Policy.PolicyDefinitionID)
				}
				if restriction.Policy.PolicyAssignmentID != nil {
					entry.PolicyAssignment = parsePolicyRef(log, *restriction.Policy.PolicyAssignmentID)
				}
			}
			out = append(out, entry)
		}
	}
	return out
}

func parsePolicyRef(log *logrus.Entry, id string) *PolicyRef {
	if id == "" {
		return nil
	}
	parsed, err := arm.ParseResourceID(id)
	if err != nil {
		log.Errorf("failed to parse policy resource id %q: %v", id, err)
		return &PolicyRef{Name: id}
	}
	return &PolicyRef{
		Name:              parsed.Name,
		SubscriptionID:    parsed.SubscriptionID,
		ResourceGroupName: parsed.ResourceGroupName,
	}
}
