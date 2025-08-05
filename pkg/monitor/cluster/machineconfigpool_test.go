package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitMachineConfigPoolUnmanagedNodes(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		objects []client.Object
		expect  int64
	}{
		{
			name:   "Too Many Nodes",
			expect: 2,
			objects: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-1",
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-config-pool",
					},
					Status: mcv1.MachineConfigPoolStatus{
						MachineCount: 0,
					},
				},
			},
		},
		{
			name:   "Too Few Nodes",
			expect: -1,
			objects: []client.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-config-pool-1",
					},
					Status: mcv1.MachineConfigPoolStatus{
						MachineCount: 1,
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-config-pool-2",
					},
					Status: mcv1.MachineConfigPoolStatus{
						MachineCount: 1,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			_, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithObjects(tt.objects...).
				Build())

			mon := &Monitor{
				log:          log,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
			}

			m.EXPECT().EmitGauge("machineconfigpool.unmanagednodescount", tt.expect, map[string]string{})

			err := mon.emitMachineConfigPoolUnmanagedNodeCounts(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
