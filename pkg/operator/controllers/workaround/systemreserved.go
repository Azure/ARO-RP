package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	v1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"

	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type systemreserved struct {
	mcocli       mcoclient.Interface
	dh           dynamichelper.Interface
	log          *logrus.Entry
	versionFixed *version.Version
}

const (
	hardEviction                = "500Mi"
	labelName                   = "aro.openshift.io/limits"
	labelValue                  = ""
	kubeletConfigName           = "aro-limits"
	workerMachineConfigPoolName = "worker"
	memReserved                 = "2000Mi"
)

var (
	_ Workaround = &systemreserved{}
)

func NewSystemReserved(log *logrus.Entry, mcocli mcoclient.Interface, dh dynamichelper.Interface) *systemreserved {
	verFixed, err := version.ParseVersion("4.99.0") // TODO set this correctly when known.
	utilruntime.Must(err)

	return &systemreserved{
		mcocli:       mcocli,
		dh:           dh,
		log:          log,
		versionFixed: verFixed,
	}
}

func (sr *systemreserved) Name() string {
	return "SystemReserved fix for bz-1857446"
}

func (sr *systemreserved) IsRequired(clusterVersion *version.Version) bool {
	return clusterVersion.Lt(sr.versionFixed)
}

func (sr *systemreserved) kubeletConfig() (*mcv1.KubeletConfig, error) {
	customEvictionHard := v1beta1.DefaultEvictionHard
	customEvictionHard["memory.available"] = hardEviction
	b, err := json.Marshal(map[string]interface{}{
		"systemReserved": map[string]interface{}{
			"memory": memReserved,
		},
		"evictionHard": customEvictionHard,
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
			KubeletConfig: &runtime.RawExtension{
				Raw: b,
			},
		},
	}, nil
}

func (sr *systemreserved) Ensure(ctx context.Context) error {
	// Step 1. Add label to worker MachineConfigPool.
	// Get the worker MachineConfigPool, modify it to add a label aro.openshift.io/limits: "", and apply the modified config.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		mcp, err := sr.mcocli.MachineconfigurationV1().MachineConfigPools().Get(ctx, workerMachineConfigPoolName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if _, ok := mcp.Labels[labelName]; ok {
			// don't update if we don't need to.
			return nil
		}
		if mcp.Labels == nil {
			mcp.Labels = map[string]string{}
		}
		mcp.Labels[labelName] = labelValue

		_, err = sr.mcocli.MachineconfigurationV1().MachineConfigPools().Update(ctx, mcp, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return err
	}

	//   Step 2. Create KubeletConfig CRD with appropriate limits.
	kc, err := sr.kubeletConfig()
	if err != nil {
		return err
	}

	return sr.dh.Ensure(ctx, kc)
}

func (sr *systemreserved) Remove(ctx context.Context) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		mcp, err := sr.mcocli.MachineconfigurationV1().MachineConfigPools().Get(ctx, workerMachineConfigPoolName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if _, ok := mcp.Labels[labelName]; !ok {
			// don't update if we don't need to.
			return nil
		}
		delete(mcp.Labels, labelName)

		_, err = sr.mcocli.MachineconfigurationV1().MachineConfigPools().Update(ctx, mcp, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return err
	}
	return sr.dh.EnsureDeleted(ctx, "KubeletConfig.machineconfiguration.openshift.io/v1", "", kubeletConfigName)
}
