package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitMachineConfigPoolUnmanagedNodes_TooManyNodes(t *testing.T) {
	ctx := context.Background()

	mcocli := mcofake.NewSimpleClientset(
		&mcv1.MachineConfigPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "machine-config-pool",
			},
			Status: mcv1.MachineConfigPoolStatus{
				MachineCount: 1,
			},
		},
	)

	cli := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-0",
		},
	}, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-1",
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		mcocli: mcocli,
		m:      m,
		cli:    cli,
	}

	m.EXPECT().EmitGauge("machineconfigpool.unmanagednodescount", int64(1), map[string]string{})

	err := mon.emitMachineConfigPoolUnmanagedNodeCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmitMachineConfigPoolUnmanagedNodes_TooFewNodes(t *testing.T) {
	ctx := context.Background()

	mcocli := mcofake.NewSimpleClientset(
		&mcv1.MachineConfigPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "machine-config-pool",
			},
			Status: mcv1.MachineConfigPoolStatus{
				MachineCount: 2,
			},
		},
	)

	cli := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-0",
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		mcocli: mcocli,
		m:      m,
		cli:    cli,
	}

	m.EXPECT().EmitGauge("machineconfigpool.unmanagednodescount", int64(-1), map[string]string{})

	err := mon.emitMachineConfigPoolUnmanagedNodeCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
