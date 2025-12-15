package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	k6tv1 "kubevirt.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitCNVVirtualMachineInstanceStatuses(t *testing.T) {
	ctx := context.Background()

	vmi1 := &k6tv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmi-1",
			Namespace: "openshift-cnv",
			Labels: map[string]string{
				"os":       "fedora",
				"workload": "server",
				"flavor":   "small",
			},
		},
		Status: k6tv1.VirtualMachineInstanceStatus{
			Phase: k6tv1.Running,
			GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{
				KernelRelease: "5.14.0-284.30.1.el9_2.x86_64",
				Machine:       "x86_64",
				Name:          "Fedora Linux",
				VersionID:     "38",
			},
		},
	}

	vmi2 := &k6tv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmi-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"os":       "rhel",
				"workload": "desktop",
				"flavor":   "medium",
			},
		},
		Status: k6tv1.VirtualMachineInstanceStatus{
			Phase: k6tv1.Scheduled,
			GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{
				KernelRelease: "4.18.0-425.3.1.el8.x86_64",
				Machine:       "x86_64",
				Name:          "Red Hat Enterprise Linux",
				VersionID:     "8.7",
			},
		},
	}

	vmi3 := &k6tv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmi-3",
			Namespace: "another-namespace",
			Labels:    map[string]string{},
		},
		Status: k6tv1.VirtualMachineInstanceStatus{
			Phase:       k6tv1.Failed,
			GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{},
		},
	}

	for _, tt := range []struct {
		name          string
		objects       []client.Object
		queryLimit    int
		expectedCalls []map[string]string
	}{
		{
			name: "multiple VMIs - emit metrics for all",
			objects: []client.Object{
				vmi1,
				vmi2,
				vmi3,
			},
			queryLimit: 10,
			expectedCalls: []map[string]string{
				{
					"namespace":               "openshift-cnv",
					"name":                    "vmi-1",
					"phase":                   "Running",
					"os":                      "fedora",
					"workload":                "server",
					"flavor":                  "small",
					"guest_os_kernel_release": "5.14.0-284.30.1.el9_2.x86_64",
					"guest_os_arch":           "x86_64",
					"guest_os_name":           "Fedora Linux",
					"guest_os_version_id":     "38",
				},
				{
					"namespace":               "test-namespace",
					"name":                    "vmi-2",
					"phase":                   "Scheduled",
					"os":                      "rhel",
					"workload":                "desktop",
					"flavor":                  "medium",
					"guest_os_kernel_release": "4.18.0-425.3.1.el8.x86_64",
					"guest_os_arch":           "x86_64",
					"guest_os_name":           "Red Hat Enterprise Linux",
					"guest_os_version_id":     "8.7",
				},
				{
					"namespace":               "another-namespace",
					"name":                    "vmi-3",
					"phase":                   "Failed",
					"os":                      "",
					"workload":                "",
					"flavor":                  "",
					"guest_os_kernel_release": "",
					"guest_os_arch":           "",
					"guest_os_name":           "",
					"guest_os_version_id":     "",
				},
			},
		},
		{
			name: "single VMI",
			objects: []client.Object{
				vmi1,
			},
			queryLimit: 10,
			expectedCalls: []map[string]string{
				{
					"namespace":               "openshift-cnv",
					"name":                    "vmi-1",
					"phase":                   "Running",
					"os":                      "fedora",
					"workload":                "server",
					"flavor":                  "small",
					"guest_os_kernel_release": "5.14.0-284.30.1.el9_2.x86_64",
					"guest_os_arch":           "x86_64",
					"guest_os_name":           "Fedora Linux",
					"guest_os_version_id":     "38",
				},
			},
		},
		{
			name:          "no VMIs",
			objects:       []client.Object{},
			queryLimit:    10,
			expectedCalls: []map[string]string{},
		},
		{
			name: "pagination with limit",
			objects: []client.Object{
				vmi1,
				vmi2,
			},
			queryLimit: 1,
			expectedCalls: []map[string]string{
				{
					"namespace":               "openshift-cnv",
					"name":                    "vmi-1",
					"phase":                   "Running",
					"os":                      "fedora",
					"workload":                "server",
					"flavor":                  "small",
					"guest_os_kernel_release": "5.14.0-284.30.1.el9_2.x86_64",
					"guest_os_arch":           "x86_64",
					"guest_os_name":           "Fedora Linux",
					"guest_os_version_id":     "38",
				},
				{
					"namespace":               "test-namespace",
					"name":                    "vmi-2",
					"phase":                   "Scheduled",
					"os":                      "rhel",
					"workload":                "desktop",
					"flavor":                  "medium",
					"guest_os_kernel_release": "4.18.0-425.3.1.el8.x86_64",
					"guest_os_arch":           "x86_64",
					"guest_os_name":           "Red Hat Enterprise Linux",
					"guest_os_version_id":     "8.7",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			_, log := testlog.New()

			scheme := kruntime.NewScheme()
			_ = k6tv1.AddToScheme(scheme)

			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build())

			mon := &Monitor{
				ocpclientset: ocpclientset,
				m:            m,
				log:          log,
				queryLimit:   tt.queryLimit,
				dims:         map[string]string{}, // Initialize to match production behavior
			}

			for _, expectedLabels := range tt.expectedCalls {
				m.EXPECT().EmitGauge("cnv.virtualmachineinstance.info", int64(1), expectedLabels)
			}

			err := mon.emitCNVVirtualMachineInstanceStatuses(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
