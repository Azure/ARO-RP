package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
)

func (m *manager) validateResources(ctx context.Context) error {
	byoNSG := m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG
	ocDynamicValidator := validate.NewOpenShiftClusterDynamicValidator(
		m.log, m.env, m.doc.OpenShiftCluster, m.subscriptionDoc, m.fpAuthorizer,
	)
	err := ocDynamicValidator.Dynamic(ctx)
	if err != nil {
		return err
	}
	// If the validation found that it's no longer BYONSG, update the doc
	// TODO this very like not needed when the API change for BYONSG is introduced
	if byoNSG != m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG {
		m.log.Infof("No longer BYONSG, updating the doc's flag from %s to %s", byoNSG, m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG)
		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSGDisabled
			return nil
		})
	}
	return err
}
