package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) gatherFailureLogs(ctx context.Context) {
	for _, f := range []func(context.Context) (interface{}, error){
		m.logClusterVersion,
		m.logNodes,
		m.logClusterOperators,
		m.logIngressControllers,
		m.logAzureInformation,
	} {
		o, err := f(ctx)
		if err != nil {
			m.log.Error(err)
			continue
		}

		b, err := json.MarshalIndent(o, "", "    ")
		if err != nil {
			m.log.Error(err)
			continue
		}

		m.log.Printf("%s: %s", steps.FriendlyName(f), string(b))
	}
}

func (m *manager) logClusterVersion(ctx context.Context) (interface{}, error) {
	if m.configcli == nil {
		return nil, nil
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cv.ManagedFields = nil

	return cv, nil
}

func (m *manager) logNodes(ctx context.Context) (interface{}, error) {
	if m.kubernetescli == nil {
		return nil, nil
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range nodes.Items {
		nodes.Items[i].ManagedFields = nil
	}

	return nodes.Items, nil
}

func (m *manager) logClusterOperators(ctx context.Context) (interface{}, error) {
	if m.configcli == nil {
		return nil, nil
	}

	cos, err := m.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range cos.Items {
		cos.Items[i].ManagedFields = nil
	}

	return cos.Items, nil
}

func (m *manager) logIngressControllers(ctx context.Context) (interface{}, error) {
	if m.operatorcli == nil {
		return nil, nil
	}

	ics, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range ics.Items {
		ics.Items[i].ManagedFields = nil
	}

	return ics.Items, nil
}

type vminfo struct {
	VMID     string
	Name     string
	SKU      string
	State    string
	Statuses []mgmtcompute.InstanceViewStatus
}

func (m *manager) logAzureInformation(ctx context.Context) (interface{}, error) {
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
