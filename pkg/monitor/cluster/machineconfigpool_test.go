package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mcv1 "github.com/openshift/api/machineconfiguration/v1"
	mcofake "github.com/openshift/client-go/machineconfiguration/clientset/versioned/fake"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitMachineConfigPoolUnmanagedNodes(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name   string
		mcps   kruntime.Object
		nodes  kruntime.Object
		expect int64
	}{
		{
			name:   "Too Many Nodes",
			expect: 1,
			nodes: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-master-0",
				},
			},
			mcps: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "machine-config-pool",
				},
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount: 0,
				},
			},
		},
		{
			name:   "Too Few Nodes",
			expect: -1,
			nodes: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-master-0",
				},
			},
			mcps: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "machine-config-pool",
				},
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount: 2,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mcocli := mcofake.NewSimpleClientset(tt.mcps)
			cli := fake.NewSimpleClientset(tt.nodes)

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				mcocli: mcocli,
				m:      m,
				cli:    cli,
			}

			m.EXPECT().EmitGauge("machineconfigpool.unmanagednodescount", tt.expect, map[string]string{})

			err := mon.emitMachineConfigPoolUnmanagedNodeCounts(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
