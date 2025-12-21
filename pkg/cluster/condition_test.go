package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	ktesting "k8s.io/client-go/testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const errMustBeNilMsg = "err must be nil; condition is retried until timeout"

func marshalAzureMachineProviderStatus(t *testing.T, status *machinev1beta1.AzureMachineProviderStatus) *runtime.RawExtension {
	buf, _ := json.Marshal(status)
	return &runtime.RawExtension{
		Raw: buf,
	}
}

func TestOperatorConsoleExists(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		consoleName string
		want        bool
	}{
		{
			name: "Can't get operator console",
		},
		{
			name:        "Operator console exists",
			consoleName: consoleConfigResourceName,
			want:        true,
		},
	} {
		m := &manager{
			operatorcli: operatorfake.NewSimpleClientset(&operatorv1.Console{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.consoleName,
				},
			}),
		}
		ready, err := m.operatorConsoleExists(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
	}
}

func TestMinimumWorkerNodesReady(t *testing.T) {
	ctx := context.Background()
	const phaseFailed = "Failed"

	for _, tt := range []struct {
		name           string
		readyCondition corev1.ConditionStatus
		nodeLabels     map[string]string
		want           bool
	}{
		{
			name: "Can't get nodes",
		},
		{
			name:           "Non-worker nodes ready, but not enough workers",
			readyCondition: corev1.ConditionTrue,
		},
		{
			name: "Not enough worker nodes ready",
			nodeLabels: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			readyCondition: corev1.ConditionFalse,
		},
		{
			name:           "Min worker nodes ready",
			readyCondition: corev1.ConditionTrue,
			nodeLabels: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			want: true,
		},
	} {
		m := &manager{
			log: logrus.NewEntry(logrus.StandardLogger()),
			maocli: machinefake.NewSimpleClientset(
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "node1",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							"machine.openshift.io/cluster-api-machine-role": "worker",
							"machine.openshift.io/cluster-api-machine-type": "worker",
						},
					},
					Status: machinev1beta1.MachineStatus{
						Phase:          pointerutils.ToPtr(phaseRunning),
						ProviderStatus: marshalAzureMachineProviderStatus(t, &machinev1beta1.AzureMachineProviderStatus{}),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "node2",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							"machine.openshift.io/cluster-api-machine-role": "worker",
							"machine.openshift.io/cluster-api-machine-type": "worker",
						},
					},
					Status: machinev1beta1.MachineStatus{
						Phase:          pointerutils.ToPtr(phaseRunning),
						ProviderStatus: marshalAzureMachineProviderStatus(t, &machinev1beta1.AzureMachineProviderStatus{}),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "node3",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							"machine.openshift.io/cluster-api-machine-role": "worker",
							"machine.openshift.io/cluster-api-machine-type": "worker",
						},
					},
					Status: machinev1beta1.MachineStatus{
						Phase:          pointerutils.ToPtr(phaseFailed),
						ProviderStatus: marshalAzureMachineProviderStatus(t, &machinev1beta1.AzureMachineProviderStatus{}),
					},
				},
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "node4-has-no-status",
						Namespace: "openshift-machine-api",
						Labels: map[string]string{
							"machine.openshift.io/cluster-api-machine-role": "worker",
							"machine.openshift.io/cluster-api-machine-type": "worker",
						},
					},
				},
				testMachine(t, "openshift-machine-api", "master1", &machinev1beta1.AzureMachineProviderSpec{}),
				testMachine(t, "openshift-machine-api", "master2", &machinev1beta1.AzureMachineProviderSpec{}),
			),
			kubernetescli: fake.NewSimpleClientset(&corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "node1",
							Labels: tt.nodeLabels,
						},
						Status: corev1.NodeStatus{
							Conditions: []corev1.NodeCondition{
								{
									Type:   corev1.NodeReady,
									Status: tt.readyCondition,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "node2",
							Labels: tt.nodeLabels,
						},
						Status: corev1.NodeStatus{
							Conditions: []corev1.NodeCondition{
								{
									Type:   corev1.NodeReady,
									Status: tt.readyCondition,
								},
							},
						},
					},
				},
			}),
		}
		ready, err := m.minimumWorkerNodesReady(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(tt.name, ready)
		}
	}
}

func TestClusterVersionReady(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name               string
		version            string
		availableCondition configv1.ConditionStatus
		want               bool
	}{
		{
			name: "Can't get cluster version",
		},
		{
			name:               "Cluster version not ready yet",
			version:            "version",
			availableCondition: configv1.ConditionFalse,
		},
		{
			name:               "Cluster version ready",
			version:            "version",
			availableCondition: configv1.ConditionTrue,
			want:               true,
		},
	} {
		m := &manager{
			configcli: configfake.NewSimpleClientset(&configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.version,
				},
				Status: configv1.ClusterVersionStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorAvailable,
							Status: tt.availableCondition,
						},
					},
				},
			}),
		}
		ready, err := m.clusterVersionReady(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
	}
}

func TestAroCredentialsRequestReconciled(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name          string
		kubernetescli func() *fake.Clientset
		dynamiccli    func() *dynamicfake.FakeDynamicClient
		spp           *api.ServicePrincipalProfile
		want          bool
		wantErrMsg    string
	}{
		{
			name: "Cluster service principal has not changed",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecret",
			},
			want: true,
		},
		{
			name: "CredentialsRequest not found",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			want: false,
		},
		{
			name: "Encounter some other error getting the CredentialsRequest",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				dynamiccli := dynamicfake.NewSimpleDynamicClient(scheme.Scheme)
				dynamiccli.PrependReactor("get", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &cloudcredentialv1.CredentialsRequest{}, errors.New("Couldn't get CredentialsRequest for arbitrary reason")
				})
				return dynamiccli
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			want:       false,
			wantErrMsg: "Couldn't get CredentialsRequest for arbitrary reason",
		},
		{
			name: "CredentialsRequest is missing status.lastSyncTimestamp",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				cr := cloudcredentialv1.CredentialsRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "openshift-azure-operator",
						Namespace: "openshift-cloud-credential-operator",
					},
				}
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme, &cr)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			want:       false,
			wantErrMsg: "unable to access status.lastSyncTimestamp of openshift-azure-operator CredentialsRequest",
		},
		{
			name: "CredentialsRequest was last synced 10 minutes ago (too long)",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				timestamp := metav1.NewTime(time.Now().Add(-10 * time.Minute))
				cr := cloudcredentialv1.CredentialsRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "openshift-azure-operator",
						Namespace: "openshift-cloud-credential-operator",
					},
					Status: cloudcredentialv1.CredentialsRequestStatus{
						LastSyncTimestamp: &timestamp,
					},
				}
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme, &cr)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			want: false,
		},
		{
			name: "CredentialsRequest was last synced 10 seconds ago",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				timestamp := metav1.NewTime(time.Now().Add(-10 * time.Second))
				cr := cloudcredentialv1.CredentialsRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "openshift-azure-operator",
						Namespace: "openshift-cloud-credential-operator",
					},
					Status: cloudcredentialv1.CredentialsRequestStatus{
						LastSyncTimestamp: &timestamp,
					},
				}
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme, &cr)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				log:           logrus.NewEntry(logrus.StandardLogger()),
				kubernetescli: tt.kubernetescli(),
				dynamiccli:    tt.dynamiccli(),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ServicePrincipalProfile: tt.spp,
						},
					},
				},
			}

			result, err := m.aroCredentialsRequestReconciled(ctx)
			if result != tt.want {
				t.Errorf("Result was %v, wanted %v", result, tt.want)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
		})
	}
}

func TestHaveClusterOperatorsSettled(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                     string
		apiserverConditions      []configv1.ClusterOperatorStatusCondition
		kubeControllerConditions []configv1.ClusterOperatorStatusCondition
		want                     bool
	}{
		{
			name: "APIServer Available, not progressing, KCM OK - ready",
			apiserverConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			kubeControllerConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			want: true,
		},
		{
			name: "APIServer OK, KCM degraded - not ready",
			apiserverConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			kubeControllerConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionTrue,
				},
			},
			want: false,
		},
		{
			name: "APIServer Available but still progressing, KCM OK - not ready",
			apiserverConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			kubeControllerConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			want: false,
		},
		{
			name: "APIServer Available but degraded, KCM OK - not ready",
			apiserverConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionTrue,
				},
			},
			kubeControllerConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			want: false,
		},
		{
			name: "APIServer Not available, KCM OK - not ready",
			apiserverConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			kubeControllerConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionFalse,
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{
				&configv1.ClusterOperator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-apiserver",
					},
					Status: configv1.ClusterOperatorStatus{
						Conditions: tt.apiserverConditions,
					},
				},
				&configv1.ClusterOperator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-controller-manager",
					},
					Status: configv1.ClusterOperatorStatus{
						Conditions: tt.kubeControllerConditions,
					},
				},
				// this being degraded should not affect the APIServer
				&configv1.ClusterOperator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-widget-operator",
					},
					Status: configv1.ClusterOperatorStatus{
						Conditions: []configv1.ClusterOperatorStatusCondition{
							{
								Type:   configv1.OperatorAvailable,
								Status: configv1.ConditionFalse,
							},
							{
								Type:   configv1.OperatorProgressing,
								Status: configv1.ConditionTrue,
							},
							{
								Type:   configv1.OperatorDegraded,
								Status: configv1.ConditionTrue,
							},
						},
					},
				},
			}
			_, log := testlog.New()
			ch := clienthelper.NewWithClient(log, clientfake.
				NewClientBuilder().
				WithObjects(objects...).
				Build())

			m := &manager{
				log:              log,
				kubeClientHelper: ch,
			}

			result, err := m.clusterOperatorsHaveSettled(ctx)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("Result was %v, wanted %v", result, tt.want)
			}
		})
	}
}
