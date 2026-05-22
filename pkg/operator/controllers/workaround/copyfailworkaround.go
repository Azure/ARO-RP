package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type copyfailworkaround struct {
	log *logrus.Entry
	ch  clienthelper.Interface
}

var _ Workaround = &copyfailworkaround{}

var copyfailFixedPatchVersions = map[string]version.Version{
	"4.21": version.NewVersion(4, 21, 14),
	"4.20": version.NewVersion(4, 20, 21),
	"4.19": version.NewVersion(4, 19, 30),
	"4.18": version.NewVersion(4, 18, 40),
	"4.17": version.NewVersion(4, 17, 53),
	"4.16": version.NewVersion(4, 16, 61),
	"4.15": version.NewVersion(4, 15, 64),
	"4.14": version.NewVersion(4, 14, 65),
	"4.13": version.NewVersion(4, 13, 66),
	"4.12": version.NewVersion(4, 12, 89),
}

func NewCopyFailWorkaround(log *logrus.Entry, client client.Client) *copyfailworkaround {
	ch := clienthelper.NewWithClient(log, client)
	return &copyfailworkaround{log: log, ch: ch}
}

// IsRequired implements [Workaround].
func (a *copyfailworkaround) IsRequired(ctx context.Context, clusterVersion version.Version, cluster *v1alpha1.Cluster) (bool, error) {
	enabled := cluster.Spec.OperatorFlags.GetSimpleBoolean(operator.CopyFailWorkaroundEnabled)
	if !enabled {
		return false, nil
	}

	if clusterVersion.Gt(version.NewVersion(4, 22, 0)) {
		return false, nil
	}

	clusterMinorVersion := clusterVersion.MinorVersion()
	if fixedPatchVersion, ok := copyfailFixedPatchVersions[clusterMinorVersion]; ok {
		return clusterVersion.Lt(fixedPatchVersion), nil
	}

	return true, nil
}

// Ensure implements [Workaround].
func (a *copyfailworkaround) Ensure(ctx context.Context) error {
	return a.ch.Ensure(ctx, makeMachineConfig("master"))
}

// Name implements [Workaround].
func (a *copyfailworkaround) Name() string {
	return "workaround for CVE-2026-31431 ('copy fail') on control plane"
}

// Remove implements [Workaround].
func (a *copyfailworkaround) Remove(ctx context.Context) error {
	return a.ch.EnsureDeleted(
		ctx,
		mcv1.GroupVersion.WithKind("MachineConfig"),
		types.NamespacedName{Name: "99-master-disable-algif-aead"},
	)
}

func makeMachineConfig(role string) *mcv1.MachineConfig {
	return &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-disable-algif-aead", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			KernelArguments: []string{"initcall_blacklist=algif_aead_init"},
		},
	}
}
