package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"

	azureproviderv1beta1 "github.com/openshift/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
	clusterapi "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type systemreserved struct {
	mcocli       mcoclient.Interface
	clustercli   clusterapi.Interface
	dh           dynamichelper.DynamicHelper
	log          *logrus.Entry
	versionFixed *version.Version
}

const (
	labelName                   = "aro.openshift.io/limits"
	labelValue                  = ""
	kubeletConfigName           = "aro-limits"
	workerMachineSetsNamespace  = "openshift-machine-api"
	workerMachineConfigPoolName = "worker"
)

var (
	_          Workaround = &systemreserved{}
	vmCapacity            = map[string]map[string]int{
		"Standard_D16as_v4": {"MemGB": 64, "vCPUs": 16},
		"Standard_D16s_v3":  {"MemGB": 64, "vCPUs": 16},
		"Standard_D2s_v3":   {"MemGB": 8, "vCPUs": 2},
		"Standard_D32as_v4": {"MemGB": 128, "vCPUs": 32},
		"Standard_D32s_v3":  {"MemGB": 128, "vCPUs": 32},
		"Standard_D4as_v4":  {"MemGB": 16, "vCPUs": 4},
		"Standard_D4s_v3":   {"MemGB": 16, "vCPUs": 4},
		"Standard_D8as_v4":  {"MemGB": 32, "vCPUs": 8},
		"Standard_D8s_v3":   {"MemGB": 32, "vCPUs": 8},
		"Standard_E16s_v3":  {"MemGB": 128, "vCPUs": 16},
		"Standard_E32s_v3":  {"MemGB": 256, "vCPUs": 32},
		"Standard_E4s_v3":   {"MemGB": 32, "vCPUs": 4},
		"Standard_E8s_v3":   {"MemGB": 64, "vCPUs": 8},
		"Standard_F16s_v2":  {"MemGB": 32, "vCPUs": 16},
		"Standard_F32s_v2":  {"MemGB": 64, "vCPUs": 32},
		"Standard_F4s_v2":   {"MemGB": 8, "vCPUs": 4},
		"Standard_F8s_v2":   {"MemGB": 16, "vCPUs": 8},
	}
	// convert total memory on the VM to amount to reserve
	memReserved = map[int]string{
		8:   "1800Mi",
		16:  "2600Mi",
		32:  "3560Mi",
		64:  "5480Mi",
		128: "9320Mi",
		256: "11880Mi",
	}
	// convert vCPUs to millicores to reserve
	cpuReserved = map[int]string{
		2:  "500m",
		4:  "500m",
		8:  "500m",
		16: "500m",
		32: "500m",
		64: "750m",
	}
)

func NewSystemReserved(log *logrus.Entry, mcocli mcoclient.Interface, clustercli clusterapi.Interface, dh dynamichelper.DynamicHelper) *systemreserved {
	utilruntime.Must(mcv1.AddToScheme(scheme.Scheme))
	verFixed, err := version.ParseVersion("4.99.0") // TODO set this correctly when known.
	utilruntime.Must(err)

	return &systemreserved{
		mcocli:       mcocli,
		clustercli:   clustercli,
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

func (sr *systemreserved) kubeletConfig(vmSize string) (*unstructured.Unstructured, error) {
	cap, ok := vmCapacity[vmSize]
	if !ok {
		return nil, fmt.Errorf("vmSize %s not valid", vmSize)
	}

	kubeletConfig := map[string]map[string]string{
		"systemReserved": {
			"memory": memReserved[cap["MemGB"]],
			"cpu":    cpuReserved[cap["vCPUs"]],
		},
	}
	cfgJSON, err := json.Marshal(kubeletConfig)
	if err != nil {
		return nil, err
	}

	kc := &mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kubeletConfigName,
			Labels: map[string]string{labelName: labelValue},
		},
		Spec: mcv1.KubeletConfigSpec{
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{labelName: labelValue},
			},
			KubeletConfig: &runtime.RawExtension{Raw: cfgJSON},
		},
	}
	un := &unstructured.Unstructured{}
	err = scheme.Scheme.Convert(kc, un, nil)
	if err != nil {
		return nil, err
	}
	return un, nil
}

func (sr *systemreserved) vmSize() (string, error) {
	machinesets, err := sr.clustercli.MachineV1beta1().MachineSets(workerMachineSetsNamespace).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	prevVMSize := ""
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Template.Spec.ProviderSpec.Value == nil {
			return "", fmt.Errorf("provider spec is missing in the machine set %q", machineset.Name)
		}

		o, _, err := scheme.Codecs.UniversalDeserializer().Decode(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw, nil, nil)
		if err != nil {
			return "", err
		}

		machineProviderSpec, ok := o.(*azureproviderv1beta1.AzureMachineProviderSpec)
		if !ok {
			// This should never happen: codecs uses scheme that has only one registered type
			// and if something is wrong with the provider spec - decoding should fail
			return "", fmt.Errorf("failed to read provider spec from the machine set %q: %T", machineset.Name, o)
		}

		if prevVMSize == "" {
			prevVMSize = machineProviderSpec.VMSize
		} else if prevVMSize != machineProviderSpec.VMSize {
			return "Standard_D2s_v3", nil // there are a mix of vmSizes, just try something sensible.
		}
	}
	return prevVMSize, nil
}

func (sr *systemreserved) Ensure() error {
	// Step 1. Add label to worker MachineConfigPool.
	// Get the worker MachineConfigPool, modify it to add a label bz-1857446: hotfixed, and apply the modified config.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		mcp, err := sr.mcocli.MachineconfigurationV1().MachineConfigPools().Get(workerMachineConfigPoolName, metav1.GetOptions{})
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

		_, err = sr.mcocli.MachineconfigurationV1().MachineConfigPools().Update(mcp)
		return err
	})
	if err != nil {
		return err
	}

	vmSize, err := sr.vmSize()
	if err != nil {
		return err
	}

	//   Step 2. Create KubeletConfig CRD with appropriate limits.
	un, err := sr.kubeletConfig(vmSize)
	if err != nil {
		return err
	}

	return sr.dh.Ensure(un)
}

func (sr *systemreserved) Remove() error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		mcp, err := sr.mcocli.MachineconfigurationV1().MachineConfigPools().Get(workerMachineConfigPoolName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if _, ok := mcp.Labels[labelName]; !ok {
			// don't update if we don't need to.
			return nil
		}
		delete(mcp.Labels, labelName)

		_, err = sr.mcocli.MachineconfigurationV1().MachineConfigPools().Update(mcp)
		return err
	})
	if err != nil {
		return err
	}
	return sr.dh.Delete("KubeletConfig.machineconfiguration.openshift.io/v1", "", kubeletConfigName)
}
