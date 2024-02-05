package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/base64"
	"strings"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) LogAzureInformation(ctx context.Context) (interface{}, error) {
	if m.virtualMachines == nil {
		return nil, nil
	}

	items := make([]interface{}, 0)
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vms, err := m.virtualMachines.List(ctx, resourceGroupName)
	if err != nil {
		items = append(items, err)
		return items, nil
	}

	consoleURIs := make([][]string, 0)

	for _, v := range vms {
		items = append(items, v)
		if v.InstanceView != nil && v.InstanceView.BootDiagnostics != nil && v.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI != nil {
			consoleURIs = append(consoleURIs, []string{*v.Name, *v.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI})
		}
	}

	blob, err := m.storage.BlobService(ctx, resourceGroupName, "cluster"+m.doc.OpenShiftCluster.Properties.StorageSuffix, mgmtstorage.R, mgmtstorage.SignedResourceTypesO)
	if err != nil {
		items = append(items, err)
		return items, nil
	}

	for _, i := range consoleURIs {
		parts := strings.Split(i[1], "/")

		c := blob.GetContainerReference(parts[1])
		b := c.GetBlobReference(parts[2])

		rc, err := b.Get(nil)
		if err != nil {
			items = append(items, err)
			continue
		}
		defer rc.Close()

		logForVM := m.log.WithField("failedRoleInstance", i[0])

		b64Reader := base64.NewDecoder(base64.StdEncoding, rc)
		scanner := bufio.NewScanner(b64Reader)
		for scanner.Scan() {
			logForVM.Info(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			items = append(items, err)
		}
	}

	return items, nil
}
