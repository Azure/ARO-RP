package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
)

func DeletePrivateDNSVNetLinks(ctx context.Context, log *logrus.Entry, vNetLinksClient privatedns.VirtualNetworkLinksClient, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	if vNetLinksClient == nil {
		return errors.New("vNetLinksClient is nil")
	}

	vNetLinks, err := vNetLinksClient.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, vNetLink := range vNetLinks {
		name := *vNetLink.Name
		err = arm.RetryableDelete(ctx, func() error {
			return vNetLinksClient.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, name, "")
		}, log, "deleting private DNS VNet link "+name)
		if err != nil {
			return err
		}
	}

	return nil
}
