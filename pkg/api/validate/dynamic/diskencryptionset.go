package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (dv *dynamic) ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("ValidateDiskEncryptionSet")

	// It is very likely that master and worker profiles use the same
	// disk encryption set, so to optimise we only validate unique ones.
	// We maintain the slice of ids separately to the map to have stable
	// validation order because iteration order for maps is not stable.
	uniqueIds := map[string]struct{}{}
	ids := []string{}
	paths := []string{}
	if oc.Properties.MasterProfile.DiskEncryptionSetID != "" {
		uniqueIds[strings.ToLower(oc.Properties.MasterProfile.DiskEncryptionSetID)] = struct{}{}
		ids = append(ids, oc.Properties.MasterProfile.DiskEncryptionSetID)
		paths = append(paths, "properties.masterProfile.diskEncryptionSetId")
	}
	for i, wp := range oc.Properties.WorkerProfiles {
		if wp.DiskEncryptionSetID != "" {
			lowercasedId := strings.ToLower(wp.DiskEncryptionSetID)
			if _, ok := uniqueIds[lowercasedId]; ok {
				continue
			}

			uniqueIds[lowercasedId] = struct{}{}
			ids = append(ids, wp.DiskEncryptionSetID)
			paths = append(paths, fmt.Sprintf("properties.workerProfiles[%d].diskEncryptionSetId", i))
		}
	}

	for i, id := range ids {
		r, err := azure.ParseResourceID(id)
		if err != nil {
			return err
		}

		err = dv.validateDiskEncryptionSetPermissions(ctx, &r, paths[i])
		if err != nil {
			return err
		}

		err = dv.validateDiskEncryptionSetLocation(ctx, &r, oc.Location, paths[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *dynamic) validateDiskEncryptionSetPermissions(ctx context.Context, desr *azure.Resource, path string) error {
	dv.log.Print("validateDiskEncryptionSetPermissions")

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err := dv.validateActions(ctx, desr, []string{
		"Microsoft.Compute/diskEncryptionSets/read",
	})

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, path, "The %s service principal does not have Reader permission on disk encryption set '%s'.", dv.authorizerType, desr.String())
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedDiskEncryptionSet, path, "The disk encryption set '%s' could not be found.", desr.String())
	}

	return err
}

func (dv *dynamic) validateDiskEncryptionSetLocation(ctx context.Context, desr *azure.Resource, location, path string) error {
	dv.log.Print("validateDiskEncryptionSetLocation")

	des, err := dv.diskEncryptionSets.Get(ctx, desr.ResourceGroup, desr.ResourceName)
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusNotFound {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedDiskEncryptionSet, path, "The disk encryption set '%s' could not be found.", desr.String())
		}
		return err
	}

	if !strings.EqualFold(*des.Location, location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedDiskEncryptionSet, "", "The disk encryption set location '%s' must match the cluster location '%s'.", *des.Location, location)
	}

	return nil
}
