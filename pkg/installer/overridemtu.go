package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/installer/pkg/asset/ignition"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/machines/machineconfig"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/cluster/graph"
)

const (
	IgnFilePath = "/etc/NetworkManager/dispatcher.d/30-eth0-mtu-3900"
	IgnFileData = `#!/bin/bash

if [ "$1" == "eth0" ] && [ "$2" == "up" ]; then
    ip link set $1 mtu 3900
fi`
)

func newMTUIgnitionFile() types.File {
	return ignition.FileFromString(IgnFilePath, "root", 0555, IgnFileData)
}

func newMTUMachineConfigIgnitionFile(role string) (types.File, error) {
	mtuIgnitionConfig := types.Config{
		Ignition: types.Ignition{
			Version: types.MaxVersion.String(),
		},
		Storage: types.Storage{
			Files: []types.File{
				newMTUIgnitionFile(),
			},
		},
	}

	rawExt, err := ignition.ConvertToRawExtension(mtuIgnitionConfig)
	if err != nil {
		return types.File{}, err
	}

	mtuMachineConfig := &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-mtu", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: rawExt,
		},
	}

	configs := []*mcv1.MachineConfig{mtuMachineConfig}
	manifests, err := machineconfig.Manifests(configs, role, "/opt/openshift/openshift")
	if err != nil {
		return types.File{}, err
	}

	return ignition.FileFromBytes(manifests[0].Filename, "root", 0644, manifests[0].Data), nil
}

func (m *manager) overrideEthernetMTU(g graph.Graph) error {
	bootstrap := g.Get(&bootstrap.Bootstrap{}).(*bootstrap.Bootstrap)

	// Override MTU on the bootstrap node itself, so cluster-network-operator
	// gets an appropriate default MTU for OpenshiftSDN or OVNKubernetes when
	// it first starts up on the bootstrap node.

	ignitionFile := newMTUIgnitionFile()
	bootstrap.Config.Storage.Files = append(bootstrap.Config.Storage.Files, ignitionFile)

	// Then add the following MachineConfig manifest files to the bootstrap
	// node's Ignition config:
	//
	// /opt/openshift/openshift/99_openshift-machineconfig_99-master-mtu.yaml
	// /opt/openshift/openshift/99_openshift-machineconfig_99-worker-mtu.yaml

	ignitionFile, err := newMTUMachineConfigIgnitionFile("master")
	if err != nil {
		return err
	}
	bootstrap.Config.Storage.Files = append(bootstrap.Config.Storage.Files, ignitionFile)

	ignitionFile, err = newMTUMachineConfigIgnitionFile("worker")
	if err != nil {
		return err
	}
	bootstrap.Config.Storage.Files = append(bootstrap.Config.Storage.Files, ignitionFile)

	data, err := ignition.Marshal(bootstrap.Config)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal Ignition config")
	}
	bootstrap.File.Data = data

	return nil
}
