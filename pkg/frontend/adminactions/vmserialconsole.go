package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) VMSerialConsole(ctx context.Context, w http.ResponseWriter,
	log *logrus.Entry, vmName string) error {

	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	vm, err := a.virtualMachines.Get(ctx, clusterRGName, vmName, mgmtcompute.InstanceView)
	if err != nil {
		return err
	}

	if vm.InstanceView == nil || vm.InstanceView.BootDiagnostics == nil {
		return fmt.Errorf("BootDiagnostics not enabled on %s, serial log is not available", vmName)
	}

	u, err := url.Parse(*vm.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI)
	if err != nil {
		return err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 3 {
		return fmt.Errorf("serialConsoleLogBlobURI has %d parts, expected 3", len(parts))
	}

	t := time.Now().UTC().Truncate(time.Second)
	res, err := a.storageAccounts.ListAccountSAS(
		ctx, clusterRGName, "cluster"+a.oc.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
			Services:               mgmtstorage.B,
			ResourceTypes:          mgmtstorage.SignedResourceTypesO,
			Permissions:            mgmtstorage.R,
			Protocols:              mgmtstorage.HTTPS,
			SharedAccessStartTime:  &date.Time{Time: t},
			SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
		})
	if err != nil {
		return err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return err
	}

	blobService := azstorage.NewAccountSASClient(
		"cluster"+a.oc.Properties.StorageSuffix, v, (*a.env.Environment()).Environment).GetBlobService()

	c := blobService.GetContainerReference(parts[1])

	b := c.GetBlobReference(parts[2])

	rc, err := b.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	w.Header().Add("Content-Type", "text/plain")

	_, err = io.Copy(w, rc)
	return err
}
