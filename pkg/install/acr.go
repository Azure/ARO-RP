package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	pkgacrtoken "github.com/Azure/ARO-RP/pkg/util/acrtoken"
)

func (i *Installer) ensureAcrToken(ctx context.Context) error {
	acrtoken, err := pkgacrtoken.NewManager(i.env, i.localFPAuthorizer)
	if err != nil {
		return err
	}

	rp := acrtoken.GetRegistryProfile(i.doc.OpenShiftCluster)
	if rp == nil {
		// 1. choose a name and establish the intent to create a token with
		// that name
		rp = acrtoken.NewRegistryProfile(i.doc.OpenShiftCluster)

		i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			acrtoken.PutRegistryProfile(doc.OpenShiftCluster, rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	if rp.Password == "" {
		// 2. ensure a token with the chosen name exists, generate a
		// password for it and store it in the database
		password, err := acrtoken.EnsureTokenAndPassword(ctx, rp)
		if err != nil {
			return err
		}

		rp.Password = api.SecureString(password)

		i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			acrtoken.PutRegistryProfile(doc.OpenShiftCluster, rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
