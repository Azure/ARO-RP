package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const errMustBeNilMsg = "err must be nil; condition is retried until timeout"

func TestBootstrapConfigMapReady(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name               string
		configMapName      string
		configMapNamespace string
		configMapStatus    string
		env                func() env.Interface
		want               bool
	}{
		{
			name: "Can't get config maps for kube-system namespace",
			env: func() env.Interface {
				env := mock_env.NewMockInterface(controller)
				env.EXPECT().IsLocalDevelopmentMode().Return(true)
				return env
			},
		},
		{
			name:               "Can't get bootstrap config map",
			configMapNamespace: "kube-system",
			env: func() env.Interface {
				env := mock_env.NewMockInterface(controller)
				env.EXPECT().IsLocalDevelopmentMode().Return(true)
				return env
			},
		},
		{
			name:               "Status not complete",
			configMapName:      "bootstrap",
			configMapNamespace: "kube-system",
			env: func() env.Interface {
				return mock_env.NewMockInterface(controller)
			},
		},
		{
			name:               "Bootstrap config map is ready",
			configMapName:      "bootstrap",
			configMapNamespace: "kube-system",
			configMapStatus:    "complete",
			env: func() env.Interface {
				return mock_env.NewMockInterface(controller)
			},
			want: true,
		},
	} {
		m := &manager{
			log: logrus.NewEntry(logrus.StandardLogger()),
			env: tt.env(),
			kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.configMapName,
					Namespace: tt.configMapNamespace,
				},
				Data: map[string]string{
					"status": tt.configMapStatus,
				},
			}),
		}
		ready, err := m.bootstrapConfigMapReady(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
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
			consoleName: consoleapi.ConfigResourceName,
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

func TestIsOperatorAvailable(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		availableCondition   configv1.ConditionStatus
		progressingCondition configv1.ConditionStatus
		want                 bool
	}{
		{
			name:                 "Available && Progressing; not available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "Available && !Progressing; available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionFalse,
			want:                 true,
		},
		{
			name:                 "!Available && Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "!Available && !Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionFalse,
		},
	} {
		operator := &configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: tt.availableCondition,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: tt.progressingCondition,
					},
				},
			},
		}
		available := isOperatorAvailable(operator)
		if available != tt.want {
			t.Error(available)
		}
	}
}

func TestMinimumWorkerNodesReady(t *testing.T) {
	ctx := context.Background()

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
			t.Error(ready)
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
		doc           api.OpenShiftCluster
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
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecret",
					},
				},
			},
			want: true,
		},
		{
			name: "Encounter error getting CredentialsRequest",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			dynamiccli: func() *dynamicfake.FakeDynamicClient {
				return dynamicfake.NewSimpleDynamicClient(scheme.Scheme)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecretNew",
					},
				},
			},
			want:       false,
			wantErrMsg: `credentialsrequests.cloudcredential.openshift.io "openshift-azure-operator" not found`,
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
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecretNew",
					},
				},
			},
			want:       false,
			wantErrMsg: "Unable to access status.lastSyncTimestamp of openshift-azure-operator CredentialsRequest",
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
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecretNew",
					},
				},
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
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecretNew",
					},
				},
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
					OpenShiftCluster: &tt.doc,
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
