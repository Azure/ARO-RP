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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type systemReserved struct {
	log *logrus.Entry

	client client.Client

	versionFixed *version.Version
}

var _ Workaround = &systemReserved{}

func NewSystemReserved(log *logrus.Entry, client client.Client) *systemReserved {
	verFixed, err := version.ParseVersion("4.8.0")
	utilruntime.Must(err)

	return &systemReserved{
		log:          log,
		client:       client,
		versionFixed: verFixed,
	}
}

func (sr *systemReserved) Name() string {
	return "SystemReserved fix for bz-1857446"
}

func (sr *systemReserved) IsRequired(clusterVersion *version.Version, cluster *arov1alpha1.Cluster) bool {
	return clusterVersion.Lt(sr.versionFixed)
}

func (sr *systemReserved) Ensure(ctx context.Context) error {
	sr.log.Debug("ensure systemReserved")

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
	kc := &mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeletConfigName,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, sr.client, kc, func() error {
		if kc.Labels == nil {
			kc.Labels = make(map[string]string)
		}
		kc.Labels[labelName] = labelValue

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
			return err
		}

		kc.Spec = mcv1.KubeletConfigSpec{
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{labelName: labelValue},
			},
			KubeletConfig: &kruntime.RawExtension{
				Raw: b,
			},
		}

		return nil
	})

	return err
}

func (sr *systemReserved) Remove(ctx context.Context) error {
	sr.log.Debug("remove systemReserved")
	mcp := &mcv1.MachineConfigPool{}
	err := sr.client.Get(ctx, types.NamespacedName{Name: workerMachineConfigPoolName}, mcp)
	if err != nil {
		return err
	}
	// don't update if we don't need to.
	if _, ok := mcp.Labels[labelName]; ok {
		delete(mcp.Labels, labelName)

		err = sr.client.Update(ctx, mcp)
		if err != nil {
			return err
		}
	}

	kc := &mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeletConfigName,
		},
	}
	return client.IgnoreNotFound(sr.client.Delete(ctx, kc))
}
