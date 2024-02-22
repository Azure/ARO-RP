package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) LogAzureInformation(ctx context.Context) (interface{}, error) {
	items := make([]interface{}, 0)

	if m.virtualMachines == nil {
		items = append(items, "vmclient missing")
		return items, nil
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vms, err := m.virtualMachines.List(ctx, resourceGroupName)
	if err != nil {
		items = append(items, fmt.Sprintf("vm listing error: %s", err))
		return items, nil
	}

	if len(vms) == 0 {
		items = append(items, "no vms found")
		return items, nil
	}

	consoleURIs := make([][]string, 0)
	for _, v := range vms {
		j, err := v.MarshalJSON()
		if err != nil {
			items = append(items, fmt.Sprintf("vm marshalling error: %s", err))
		} else {
			vmName := "<unknown>"
			if v.Name != nil {
				vmName = *v.Name
			}
			items = append(items, fmt.Sprintf("vm %s: %s", vmName, string(j)))
		}
		if v.VirtualMachineProperties != nil && v.InstanceView != nil && v.InstanceView.BootDiagnostics != nil && v.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI != nil {
			consoleURIs = append(consoleURIs, []string{*v.Name, *v.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI})
		}
	}

	if len(consoleURIs) == 0 {
		items = append(items, "no usable console URIs found")
		return items, nil
	}

	blob, err := m.storage.BlobService(ctx, resourceGroupName, "cluster"+m.doc.OpenShiftCluster.Properties.StorageSuffix, mgmtstorage.R, mgmtstorage.SignedResourceTypesO)
	if err != nil {
		items = append(items, fmt.Sprintf("blob storage error: %s", err))
		return items, nil
	}

	for _, i := range consoleURIs {
		rc, err := blob.Get(i[1])
		if err != nil {
			items = append(items, fmt.Sprintf("blob storage get error on %s: %s", i[0], err))
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
			items = append(items, fmt.Sprintf("blob storage scan on %s: %s", i[0], err))
		}
	}

	return items, nil
}
