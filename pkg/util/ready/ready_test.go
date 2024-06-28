package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNodeIsReady(t *testing.T) {
	for _, tt := range []struct {
		name string
		node *corev1.Node
		want bool
	}{
		{
			name: "node-is-not-ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodeDiskPressure,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodePIDPressure,
							Status: corev1.ConditionFalse,
						},
					},
				}},
			want: false,
		},
		{
			name: "node-is-ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodeDiskPressure,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodePIDPressure,
							Status: corev1.ConditionFalse,
						},
					},
				}},
			want: true,
		},
		{
			name: "node-with-no-status",
			node: &corev1.Node{},
			want: false,
		}} {
		t.Run(tt.name, func(t *testing.T) {
			if got := NodeIsReady(tt.node); tt.want != got {
				t.Fatalf("error with NodeIsReady: got %v wanted: %v", got, tt.want)
			}
		})
	}
}

func TestDaemonSetIsReady(t *testing.T) {
	for _, tt := range []struct {
		name string
		ds   *appsv1.DaemonSet
		want bool
	}{
		{
			name: "daemonset-with-no-status",
			ds:   &appsv1.DaemonSet{},
			want: false,
		},
		{
			name: "daemonset-ready",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ready-ds",
					Namespace: "default",
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: int32(6),
					DesiredNumberScheduled: int32(6),
					UpdatedNumberScheduled: int32(6),
					NumberReady:            int32(6),
					NumberAvailable:        int32(6),
				},
			},
			want: true,
		},
		{
			name: "daemonset-not-ready",
			ds: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-ready-ds",
					Namespace: "default",
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: int32(3),
					DesiredNumberScheduled: int32(2),
					UpdatedNumberScheduled: int32(4),
					NumberReady:            int32(1),
					NumberAvailable:        int32(1),
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaemonSetIsReady(tt.ds); tt.want != got {
				t.Fatalf("error with DaemonSetIsReady test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestServiceIsReady(t *testing.T) {
	for _, tt := range []struct {
		name string
		svc  *corev1.Service
		want bool
	}{
		{
			name: "service-has-no-serviceType",
			svc:  &corev1.Service{},
			want: false,
		},
		{
			name: "service-is-ready-with-type-loadbalancer",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-with-lb",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								IP:       "8.8.8.8",
								Hostname: "host",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "service-is-not-ready-with-type-loadbalancer",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-with-lb",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{},
					},
				},
			},
			want: false,
		},
		{
			name: "service-is-ready-with-type-clusterIP",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-with-clusterIP",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "8.8.8.8",
				},
			},
			want: true,
		},
		{
			name: "service-is-not-ready-with-type-clusterIP",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-with-clusterIP",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServiceIsReady(tt.svc); tt.want != got {
				t.Fatalf("error with ServiceIsReady test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDeploymentIsReady(t *testing.T) {
	specReplicas := int32(1)
	for _, tt := range []struct {
		name       string
		deployment *appsv1.Deployment
		want       bool
	}{
		{
			name:       "deployment-with-no-status",
			deployment: &appsv1.Deployment{},
			want:       false,
		},
		{
			name: "deployment-is-ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ready-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &specReplicas,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: int32(1),
					UpdatedReplicas:   int32(1),
					Replicas:          int32(1),
				},
			},
			want: true,
		},
		{
			name: "deployment-is-not-ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "not-ready-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &specReplicas,
				},
				Status: appsv1.DeploymentStatus{
					UpdatedReplicas: int32(0),
					Replicas:        int32(0),
				},
			},

			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeploymentIsReady(tt.deployment); tt.want != got {
				t.Fatalf("error with DeploymentIsReady test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestStatefulSetIsReady(t *testing.T) {
	specReplicas := int32(3)
	for _, tt := range []struct {
		name string
		ss   *appsv1.StatefulSet
		want bool
	}{
		{
			name: "statefulset-with-no-status",
			ss:   &appsv1.StatefulSet{},
			want: false,
		},
		{
			name: "statefulset-is-ready",
			ss: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: &specReplicas,
				},
				Status: appsv1.StatefulSetStatus{
					UpdatedReplicas: int32(3),
					ReadyReplicas:   int32(3),
				},
			},
			want: true,
		},
		{
			name: "statefulset-is-not-ready",
			ss: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: &specReplicas,
				},
				Status: appsv1.StatefulSetStatus{
					UpdatedReplicas: int32(1),
					ReadyReplicas:   int32(2),
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatefulSetIsReady(tt.ss); tt.want != got {
				t.Fatalf("error with StatefulSetIsReady test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestMachineConfigPoolIsReady(t *testing.T) {
	for _, tt := range []struct {
		name string
		mcp  *mcv1.MachineConfigPool
		want bool
	}{
		{
			name: "machineconfigpool-with-no-count",
			mcp:  &mcv1.MachineConfigPool{},
			want: true,
		},
		{
			name: "machineconfigpool-ready",
			mcp: &mcv1.MachineConfigPool{
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount:        int32(6),
					UpdatedMachineCount: int32(6),
					ReadyMachineCount:   int32(6),
				},
			},
			want: true,
		},
		{
			name: "machineconfigpool-not-ready",
			mcp: &mcv1.MachineConfigPool{
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount:        int32(6),
					UpdatedMachineCount: int32(3),
					ReadyMachineCount:   int32(5),
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := MachineConfigPoolIsReady(tt.mcp); tt.want != got {
				t.Fatalf("error with MachineConfigPoolIsReady test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPodIsRunning(t *testing.T) {
	for _, tt := range []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "pod-with-no-status",
			pod:  &corev1.Pod{},
			want: false,
		},
		{
			name: "pod-is-running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			want: true,
		},
		{
			name: "pod-is-not-running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := PodIsRunning(tt.pod); tt.want != got {
				t.Fatalf("error with PodIsRunning test %s: got %v wanted %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCheckDaemonSetIsRunning(t *testing.T) {
	ctx := context.Background()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-ds",
			Namespace: "default",
		},
	}
	clientset := fake.NewSimpleClientset()
	ds, err := clientset.AppsV1().DaemonSets("default").Create(ctx, ds, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating daemonset: %v", err)
	}
	_, err = CheckDaemonSetIsReady(ctx, clientset.AppsV1().DaemonSets("default"), ds.ObjectMeta.Name)()

	if err != nil {
		t.Fatalf("check daemonset is not running: %v", err)
	}
}

func TestCheckDaemonSetIsRunningNotFound(t *testing.T) {
	ctx := context.Background()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "not-found-ds",
			Namespace: "default",
		},
	}

	clientset := fake.NewSimpleClientset()
	ok, err := CheckDaemonSetIsReady(ctx, clientset.AppsV1().DaemonSets("default"), ds.ObjectMeta.Name)()

	if ok {
		t.Fatalf("check daemonset is not running: %v", err)
	}
}

func TestCheckDaemonSetIsRunningError(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewSimpleClientset()
	clientset.Fake.PrependReactor("get", "daemonsets", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, &appsv1.DaemonSet{}, errors.New("error getting daemonset")
	})
	_, err := CheckDaemonSetIsReady(ctx, clientset.AppsV1().DaemonSets("default"), "")()

	if err == nil {
		t.Fatalf("check daemonset is not ready: %v", err)
	}
}

func TestCheckPodIsRunning(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-pod",
			Namespace: "default",
		},
	}

	clientset := fake.NewSimpleClientset()
	_, err := clientset.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating pod: %v", err)
	}
	_, err = CheckPodIsRunning(ctx, clientset.CoreV1().Pods("default"), pod.ObjectMeta.Name)()
	if err != nil {
		t.Fatalf("error getting running pod: %v", err)
	}
}

func TestCheckPodIsRunningNotFound(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "not-found",
			Namespace: "default",
		},
	}

	clientset := fake.NewSimpleClientset()
	ok, err := CheckPodIsRunning(ctx, clientset.CoreV1().Pods("default"), pod.ObjectMeta.Name)()
	if ok {
		t.Fatalf("error getting running pod: %v", err)
	}
}

func TestCheckPodIsRunningError(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewSimpleClientset()
	clientset.Fake.PrependReactor("get", "pods", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, &corev1.Pod{}, errors.New("error getting pod")
	})
	_, err := CheckPodIsRunning(ctx, clientset.CoreV1().Pods("default"), "")()

	if err == nil {
		t.Fatalf("check pod is found: %v", err)
	}
}

func TestCheckDeploymentIsReady(t *testing.T) {
	ctx := context.Background()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-not-found",
			Namespace: "default",
		},
	}
	clientset := fake.NewSimpleClientset()
	_, err := clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating deployment: %v", err)
	}
	_, err = CheckDeploymentIsReady(ctx, clientset.AppsV1().Deployments("default"), deployment.ObjectMeta.Name)()

	if err != nil {
		t.Fatalf("check deployement is not ready: %v", err)
	}
}

func TestCheckDeploymentIsReadyNotFound(t *testing.T) {
	ctx := context.Background()
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-not-found",
			Namespace: "default",
		},
	}
	clientset := fake.NewSimpleClientset()
	ok, _ := CheckDeploymentIsReady(ctx, clientset.AppsV1().Deployments("default"), deployment.ObjectMeta.Name)()

	if ok {
		t.Fatalf("check deployment is found")
	}
}

func TestCheckDeploymentIsReadyError(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewSimpleClientset()
	clientset.Fake.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, &appsv1.Deployment{}, errors.New("error getting deployment")
	})
	_, err := CheckDeploymentIsReady(ctx, clientset.AppsV1().Deployments("default"), "")()

	if err == nil {
		t.Fatalf("check deployment error is: %v", err)
	}
}

func TestCheckMachineConfigPoolIsReady(t *testing.T) {
	ctx := context.Background()

	machineconfigpool := &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machineconfigpool-not-found",
		},
	}
	clientset := mcofake.NewSimpleClientset()
	_, err := clientset.MachineconfigurationV1().MachineConfigPools().Create(ctx, machineconfigpool, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating machineconfigpool: %v", err)
	}
	_, err = CheckMachineConfigPoolIsReady(ctx, clientset.MachineconfigurationV1().MachineConfigPools(), machineconfigpool.ObjectMeta.Name)()

	if err != nil {
		t.Fatalf("check machineconfigpool is not ready: %v", err)
	}
}

func TestCheckMachineConfigPoolIsReadyNotFound(t *testing.T) {
	ctx := context.Background()
	machineconfigpool := &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machineconfigpool-not-found",
		},
	}
	clientset := mcofake.NewSimpleClientset()
	ok, _ := CheckMachineConfigPoolIsReady(ctx, clientset.MachineconfigurationV1().MachineConfigPools(), machineconfigpool.ObjectMeta.Name)()

	if ok {
		t.Fatalf("check machineconfigpool is found")
	}
}

func TestCheckMachineConfigPoolIsReadyError(t *testing.T) {
	ctx := context.Background()

	clientset := mcofake.NewSimpleClientset()
	clientset.Fake.PrependReactor("get", "machineconfigpools", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, &mcv1.MachineConfigPool{}, errors.New("error getting machineconfigpool")
	})
	_, err := CheckMachineConfigPoolIsReady(ctx, clientset.MachineconfigurationV1().MachineConfigPools(), "")()

	if err == nil {
		t.Fatalf("check machineconfigpool error is: %v", err)
	}
}

func TestCheckPodsAreRunning(t *testing.T) {
	ctx := context.Background()
	labels := make(map[string]string)
	labels["app"] = "running"
	clientset := fake.NewSimpleClientset()
	clientset.Fake.PrependReactor("list", "pods", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return false, &corev1.PodList{Items: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{
				Name:      "one-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "running"},
			},
			},
		},
		}, errors.New("error listing pods")
	})
	ok, _ := CheckPodsAreRunning(ctx, clientset.CoreV1().Pods("default"), labels)()
	if !ok {
		t.Fatalf("check pods are not running: %v", ok)
	}
}
func TestCheckPodsAreReadyError(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewSimpleClientset()
	clientset.Fake.PrependReactor("list", "pods", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, &corev1.PodList{}, errors.New("error getting pods")
	})
	labels := make(map[string]string)
	labels["app"] = "running"
	_, err := CheckPodsAreRunning(ctx, clientset.CoreV1().Pods(""), labels)()

	if err == nil {
		t.Fatalf("check pod error is: %v", err)
	}
}

func TestTotalMachinesInTheMCPs(t *testing.T) {
	mcpPrefix := "99-"
	mcpSuffix := "-aro-dns"

	type testCase struct {
		name               string
		machineConfigPools []mcv1.MachineConfigPool
		want               int
		wantErr            string
	}
	testCases := []testCase{
		{
			name: "totalMachinesInTheMCPs returns ARO DNS config not found error",
			want: 0,
			machineConfigPools: []mcv1.MachineConfigPool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mcp_name",
					},
					Status: mcv1.MachineConfigPoolStatus{
						ObservedGeneration: 0,
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []corev1.ObjectReference{{Name: "non matching name"}},
						},
					},
				},
			},
			wantErr: "ARO DNS config not found in MCP mcp_name",
		},
		{
			name:    "totalMachinesInTheMCPs returns MCP is not ready error",
			want:    0,
			wantErr: "MCP mcp_name not ready",
			machineConfigPools: []mcv1.MachineConfigPool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "mcp_name",
						Generation: 1,
					},
					Status: mcv1.MachineConfigPoolStatus{
						MachineCount:        1,
						UpdatedMachineCount: 1,
						ReadyMachineCount:   0,
						ObservedGeneration:  1,
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []corev1.ObjectReference{{Name: mcpPrefix + "mcp_name" + mcpSuffix}},
						},
					},
				},
			},
		},
		{
			name: "totalMachinesInTheMCPs returns 1",
			want: 1,
			machineConfigPools: []mcv1.MachineConfigPool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "mcp_name",
						Generation: 1,
					},
					Status: mcv1.MachineConfigPoolStatus{
						MachineCount:        1,
						UpdatedMachineCount: 1,
						ReadyMachineCount:   1,
						ObservedGeneration:  1,
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []corev1.ObjectReference{
								{
									Name: mcpPrefix + "mcp_name" + mcpSuffix,
								},
							},
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name:               "totalMachinesInTheMCPs returns 0 and no error when there are no machine config pools",
			want:               0,
			machineConfigPools: nil,
			wantErr:            "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TotalMachinesInTheMCPs(tc.machineConfigPools)
			utilerror.AssertErrorMessage(t, err, tc.wantErr)

			if got != tc.want {
				t.Fatalf("expected %v, but got %v", tc.want, got)
			}
		})
	}
}

func TestMcpContainsARODNSConfig(t *testing.T) {
	mcpPrefix := "99-"
	mcpSuffix := "-aro-dns"

	type testCase struct {
		name string
		mcp  mcv1.MachineConfigPool
		want bool
	}

	testcases := []testCase{
		{
			name: "Mcp contains ARO DNS config",
			mcp: mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mcp_name",
				},
				Status: mcv1.MachineConfigPoolStatus{
					ObservedGeneration: 0,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{
							{
								Name: mcpPrefix + "mcp_name" + mcpSuffix,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Mcp does not contain ARO DNS config due to wrong source name",
			mcp: mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mcp_name",
				},
				Status: mcv1.MachineConfigPoolStatus{
					ObservedGeneration: 0,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{
							{
								Name: "non matching name",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Mcp does not contain ARO DNS config due to empty status",
			mcp: mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mcp_name",
				},
			},
			want: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := MCPContainsARODNSConfig(tc.mcp)
			if got != tc.want {
				t.Fatalf("expected %v, but got %v", tc.want, got)
			}
		})
	}
}

type fakeMcpLister struct {
	expected    *mcv1.MachineConfigPoolList
	expectedErr error
}

func (m *fakeMcpLister) List(ctx context.Context, opts metav1.ListOptions) (*mcv1.MachineConfigPoolList, error) {
	return m.expected, m.expectedErr
}

type fakeNodeLister struct {
	expected    *corev1.NodeList
	expectedErr error
}

func (m *fakeNodeLister) List(ctx context.Context, opts metav1.ListOptions) (*corev1.NodeList, error) {
	return m.expected, m.expectedErr
}

func TestSameNumberOfNodesAndMachines(t *testing.T) {
	t.Run("should return nil mcpLister error", func(t *testing.T) {
		got, err := SameNumberOfNodesAndMachines(context.Background(), nil, nil)
		wantErr := "mcpLister is nil"

		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, wantErr)
	})
	t.Run("should return nil nodeLister error", func(t *testing.T) {
		mcpLister := &fakeMcpLister{}
		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nil)
		wantErr := "nodeLister is nil"

		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, wantErr)
	})

	t.Run("should propagate error from mcpLister.List()", func(t *testing.T) {
		expectedErr := "some_error"
		mcpLister := &fakeMcpLister{expectedErr: errors.New(expectedErr)}
		nodeLister := &fakeNodeLister{}

		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nodeLister)
		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, expectedErr)
	})
	t.Run("should propagate error from TotalMachinesInTheMCPs()", func(t *testing.T) {
		expectedErr := "ARO DNS config not found in MCP mcp_name"
		badMachineConfigPools := []mcv1.MachineConfigPool{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "mcp_name"},
				Status: mcv1.MachineConfigPoolStatus{
					ObservedGeneration: 0,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{{Name: "non matching name"}},
					},
				},
			},
		}
		machineConfigPoolList := &mcv1.MachineConfigPoolList{Items: badMachineConfigPools}
		mcpLister := &fakeMcpLister{expected: machineConfigPoolList}
		nodeLister := &fakeNodeLister{}

		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nodeLister)
		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, expectedErr)
	})

	t.Run("should propagate error from nodeLister", func(t *testing.T) {
		expectedErr := "some_error"

		machineConfigPools := []mcv1.MachineConfigPool{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mcp_name",
					Generation: 1,
				},
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount:        1,
					UpdatedMachineCount: 1,
					ReadyMachineCount:   1,
					ObservedGeneration:  1,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{{Name: "99-" + "mcp_name" + "-aro-dns"}},
					},
				},
			},
		}
		machineConfigPoolList := &mcv1.MachineConfigPoolList{Items: machineConfigPools}
		mcpLister := &fakeMcpLister{expected: machineConfigPoolList}
		nodeLister := &fakeNodeLister{expectedErr: errors.New(expectedErr)}

		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nodeLister)
		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, expectedErr)
	})

	t.Run("should return true and no error", func(t *testing.T) {
		expectedErr := ""
		machineConfigPools := []mcv1.MachineConfigPool{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mcp_name",
					Generation: 1,
				},
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount:        1,
					UpdatedMachineCount: 1,
					ReadyMachineCount:   1,
					ObservedGeneration:  1,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{{Name: "99-" + "mcp_name" + "-aro-dns"}},
					},
				},
			},
		}
		machineConfigPoolList := &mcv1.MachineConfigPoolList{Items: machineConfigPools}
		mcpLister := &fakeMcpLister{expected: machineConfigPoolList}

		nodes := &corev1.NodeList{Items: []corev1.Node{{}}}
		nodeLister := &fakeNodeLister{expected: nodes}

		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nodeLister)
		if got != true {
			t.Fatalf("got %v, but wanted %v", got, true)
		}
		utilerror.AssertErrorMessage(t, err, expectedErr)
	})

	t.Run("should return false and error due to different number of total machines and nodes", func(t *testing.T) {
		expectedErr := "cluster has 3 nodes but 1 under MCPs, not removing private DNS zone"
		machineConfigPools := []mcv1.MachineConfigPool{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mcp_name",
					Generation: 1,
				},
				Status: mcv1.MachineConfigPoolStatus{
					MachineCount:        1,
					UpdatedMachineCount: 1,
					ReadyMachineCount:   1,
					ObservedGeneration:  1,
					Configuration: mcv1.MachineConfigPoolStatusConfiguration{
						Source: []corev1.ObjectReference{{Name: "99-" + "mcp_name" + "-aro-dns"}},
					},
				},
			},
		}
		machineConfigPoolList := &mcv1.MachineConfigPoolList{Items: machineConfigPools}
		mcpLister := &fakeMcpLister{expected: machineConfigPoolList}

		nodes := &corev1.NodeList{Items: []corev1.Node{{}, {}, {}}}
		nodeLister := &fakeNodeLister{expected: nodes}

		got, err := SameNumberOfNodesAndMachines(context.Background(), mcpLister, nodeLister)
		if got != false {
			t.Fatalf("got %v, but wanted %v", got, false)
		}
		utilerror.AssertErrorMessage(t, err, expectedErr)
	})
}
