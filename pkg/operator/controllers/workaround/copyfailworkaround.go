package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

func NewAlgifAEADDisable(log *logrus.Entry, client client.Client) *copyfailworkaround {
	ch := clienthelper.NewWithClient(log, client)
	return &copyfailworkaround{log: log, ch: ch}
}

// IsRequired implements [Workaround].
func (a *copyfailworkaround) IsRequired(ctx context.Context, clusterVersion version.Version, cluster *v1alpha1.Cluster) (bool, error) {
	enabled := cluster.Spec.OperatorFlags.GetSimpleBoolean(operator.CopyFailWorkaroundEnabled)
	if !enabled {
		return false, nil
	}

	// check if it is a FIPS cluster -- don't do it if there's a 99-master-fips
	mc := &mcv1.MachineConfig{}
	err := a.ch.GetOne(ctx, types.NamespacedName{Name: "99-master-fips"}, mc)
	if kerrors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
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
