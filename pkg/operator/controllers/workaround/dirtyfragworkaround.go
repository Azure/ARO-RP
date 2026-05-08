package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type dirtyfragworkaround struct {
	log *logrus.Entry
	ch  clienthelper.Interface
}

var _ Workaround = &dirtyfragworkaround{}

const ipsecModeDisabled = "Disabled"

var marshalDirtyfragIgnition = json.Marshal

func NewDirtyfragWorkaround(log *logrus.Entry, client client.Client) *dirtyfragworkaround {
	ch := clienthelper.NewWithClient(log, client)
	return &dirtyfragworkaround{log: log, ch: ch}
}

// IsRequired implements [Workaround].
func (a *dirtyfragworkaround) IsRequired(ctx context.Context, clusterVersion version.Version, cluster *v1alpha1.Cluster) (bool, error) {
	enabled := cluster.Spec.OperatorFlags.GetSimpleBoolean(operator.DirtyfragWorkaroundEnabled)
	if !enabled {
		return false, nil
	}

	network := &unstructured.Unstructured{}
	network.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.openshift.io",
		Version: "v1",
		Kind:    "Network",
	})

	err := a.ch.Get(ctx, types.NamespacedName{Name: "cluster"}, network)
	if err != nil && !kerrors.IsNotFound(err) {
		return false, fmt.Errorf("failed to get Network resource: %w", err)
	}

	if err == nil {
		ipsecConfig, found, err := unstructured.NestedMap(network.Object, "spec", "defaultNetwork", "ovnKubernetesConfig", "ipsecConfig")
		if err != nil {
			return false, fmt.Errorf("failed to parse Network resource: %w", err)
		}
		if found {
			mode, ok := ipsecConfig["mode"].(string)
			if ok && mode != ipsecModeDisabled {
				return false, nil
			}
		}
	}

	return true, nil
}

// Ensure implements [Workaround].
func (a *dirtyfragworkaround) Ensure(ctx context.Context) error {
	mc, err := makeDirtyfragMachineConfig("master")
	if err != nil {
		return err
	}

	return a.ch.Ensure(ctx, mc)
}

// Name implements [Workaround].
func (a *dirtyfragworkaround) Name() string {
	return "workaround for CVE-2026-31432 ('dirtyfrag') on control plane"
}

// Remove implements [Workaround].
func (a *dirtyfragworkaround) Remove(ctx context.Context) error {
	return a.ch.EnsureDeleted(
		ctx,
		mcv1.GroupVersion.WithKind("MachineConfig"),
		types.NamespacedName{Name: "99-master-disable-dirtyfrag"},
	)
}

func makeDirtyfragMachineConfig(role string) (*mcv1.MachineConfig, error) {
	// File content to write
	content := `install esp4 /bin/false
install esp6 /bin/false
install rxrpc /bin/false
`

	// Base64 encode the content
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	// Create Ignition config
	ignitionConfig := map[string]interface{}{
		"ignition": map[string]interface{}{
			"version": "3.2.0",
		},
		"storage": map[string]interface{}{
			"files": []map[string]interface{}{
				{
					"path":      "/etc/modprobe.d/aro-dirtyfrag-mitigation-controlplane.conf",
					"mode":      420, // 0644 in octal
					"overwrite": true,
					"contents": map[string]interface{}{
						"source": "data:text/plain;charset=utf-8;base64," + encodedContent,
					},
				},
			},
		},
	}

	// Marshal to JSON
	ignitionJSON, err := marshalDirtyfragIgnition(ignitionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dirtyfrag ignition config: %w", err)
	}

	return &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-disable-dirtyfrag", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: runtime.RawExtension{
				Raw: ignitionJSON,
			},
		},
	}, nil
}
