package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/autosizednodes"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type systemreserved struct {
	log *logrus.Entry

	client client.Client
	dh     dynamichelper.Interface

	versionFixed *version.Version
}

var _ Workaround = &systemreserved{}

func NewSystemReserved(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *systemreserved {
	verFixed, err := version.ParseVersion("4.99.0") // TODO set this correctly when known.
	utilruntime.Must(err)

	return &systemreserved{
		log:          log,
		client:       client,
		dh:           dh,
		versionFixed: verFixed,
	}
}

func (sr *systemreserved) Name() string {
	return "SystemReserved fix for bz-1857446"
}

func (sr *systemreserved) IsRequired(clusterVersion *version.Version, cluster *arov1alpha1.Cluster) bool {
	if cluster.Spec.OperatorFlags.GetSimpleBoolean(autosizednodes.ControllerEnabled) {
		return false
	}
	return clusterVersion.Lt(sr.versionFixed)
}

func (sr *systemreserved) kubeletConfig() (*mcv1.KubeletConfig, error) {
	b, err := json.Marshal(map[string]interface{}{
		"systemReserved": map[string]interface{}{
			"memory": memReserved,
		},
		"evictionHard": map[string]interface{}{
			"memory.available":  hardEviction,
			"nodefs.available":  nodeFsAvailable,
			"nodefs.inodesFree": nodeFsInodes,
			"imagefs.available": imageFs,
		},
	})
	if err != nil {
		return nil, err
	}

	return &mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kubeletConfigName,
			Labels: map[string]string{labelName: labelValue},
		},
		Spec: mcv1.KubeletConfigSpec{
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{labelName: labelValue},
			},
			KubeletConfig: &kruntime.RawExtension{
				Raw: b,
			},
		},
	}, nil
}

func (sr *systemreserved) Ensure(ctx context.Context) error {
	sr.log.Debug("ensure systemreserved")

	// Step 1. Add label to worker MachineConfigPool.
	// Get the worker MachineConfigPool, modify it to add a label aro.openshift.io/limits: "", and apply the modified config.
	mcp := &mcv1.MachineConfigPool{}
	err := sr.client.Get(ctx, types.NamespacedName{Name: workerMachineConfigPoolName}, mcp)
	if err != nil {
		return err
	}
	// don't update if we don't need to.
	if _, ok := mcp.Labels[labelName]; !ok {
		if mcp.Labels == nil {
			mcp.Labels = map[string]string{}
		}
		mcp.Labels[labelName] = labelValue

		err = sr.client.Update(ctx, mcp)
		if err != nil {
			return err
		}
	}

	//   Step 2. Create KubeletConfig CRD with appropriate limits.
	kc, err := sr.kubeletConfig()
	if err != nil {
		return err
	}

	return sr.dh.Ensure(ctx, kc)
}

func (sr *systemreserved) Remove(ctx context.Context) error {
	sr.log.Debug("remove systemreserved")
	mcp := &mcv1.MachineConfigPool{}
	err := sr.client.Get(ctx, types.NamespacedName{Name: workerMachineConfigPoolName}, mcp)
	if err != nil {
		return err
	}
	if _, ok := mcp.Labels[labelName]; !ok {
		// don't update if we don't need to.
		return nil
	}
	delete(mcp.Labels, labelName)

	err = sr.client.Update(ctx, mcp)
	if err != nil {
		return err
	}
	return sr.dh.EnsureDeleted(ctx, "KubeletConfig.machineconfiguration.openshift.io", "", kubeletConfigName)
}
