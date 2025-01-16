package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var (
	managedFields = []metav1.ManagedFieldsEntry{{Manager: "something"}}
	doc           = &api.OpenShiftClusterDocument{}
	cd            = &hivev1.ClusterDeployment{ObjectMeta: metav1.ObjectMeta{Name: "cluster", ManagedFields: managedFields}}
	cvv           = &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version", ManagedFields: managedFields}}
	master0Node   = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-aaaaa-master-0", ManagedFields: managedFields},
		Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}},
	}
	master1Node = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-aaaaa-master-1", ManagedFields: managedFields},
		Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}}},
	}
	master2Node = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-aaaaa-master-2", ManagedFields: managedFields},
		Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionUnknown}}},
	}
	aroOperator = &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "aro", ManagedFields: managedFields},
		Status: configv1.ClusterOperatorStatus{Conditions: []configv1.ClusterOperatorStatusCondition{
			{Type: configv1.OperatorAvailable, Status: configv1.ConditionTrue},
			{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse},
			{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse},
		}},
	}
	machineApiOperator = &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "machine-api", ManagedFields: managedFields},
		Status: configv1.ClusterOperatorStatus{Conditions: []configv1.ClusterOperatorStatusCondition{
			{Type: configv1.OperatorAvailable, Status: configv1.ConditionFalse},
			{Type: configv1.OperatorProgressing, Status: configv1.ConditionUnknown},
			{Type: configv1.OperatorDegraded, Status: configv1.ConditionTrue},
		}},
	}
	defaultIngressController = &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-ingress-operator", Name: "default", ManagedFields: managedFields},
		Status: operatorv1.IngressControllerStatus{Conditions: []operatorv1.OperatorCondition{
			{Type: operatorv1.OperatorStatusTypeAvailable, Status: operatorv1.ConditionTrue},
			{Type: operatorv1.OperatorStatusTypeProgressing, Status: operatorv1.ConditionFalse},
			{Type: operatorv1.OperatorStatusTypeDegraded, Status: operatorv1.ConditionFalse},
		}},
	}
	aroOperatorMasterPod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-azure-operator", Name: "aro-operator-master-aaaaaaaaa-aaaaa"}, Status: corev1.PodStatus{}}
	aroOperatorWorkerPod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-azure-operator", Name: "aro-operator-worker-bbbbbbbbb-bbbbb"}, Status: corev1.PodStatus{}}
)

func TestLogClusterDeployment(t *testing.T) {
	for _, tt := range []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		cd      *hivev1.ClusterDeployment
		want    interface{}
		wantErr string
	}{
		{
			name: "no clusterdoc returns empty",
		},
		{
			name:    "no clusterdeployment returns error",
			doc:     doc,
			wantErr: `clusterdeployments.hive.openshift.io "cluster" not found`,
		},
		{
			name: "clusterdeployment present returns clusterdeployment without managed fields",
			doc:  doc,
			cd:   cd,
			want: &hivev1.ClusterDeployment{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx := context.Background()
			_, log := testlog.New()

			mockHiveManager := mock_hive.NewMockClusterManager(controller)
			if tt.cd == nil {
				mockHiveManager.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Eq(tt.doc)).
					Return(nil, fmt.Errorf(`clusterdeployments.hive.openshift.io "cluster" not found`)).
					AnyTimes()
			} else {
				mockHiveManager.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Eq(tt.doc)).
					Return(tt.cd, nil)
			}

			m := &manager{
				log:                log,
				hiveClusterManager: mockHiveManager,
				doc:                tt.doc,
			}

			got, gotErr := m.logClusterDeployment(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogClusterVersion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		objects []kruntime.Object
		want    interface{}
		wantErr string
	}{
		{
			name:    "no cv resource returns err",
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
		{
			name:    "returns cv resource if present",
			objects: []kruntime.Object{cvv},
			want:    &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			configcli := configfake.NewSimpleClientset(tt.objects...)

			_, log := testlog.New()

			m := &manager{
				log:       log,
				configcli: configcli,
			}

			got, gotErr := m.logClusterVersion(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogNodes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		want     interface{}
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple node output and logs full node object",
			objects: []kruntime.Object{master0Node, master1Node, master2Node},
			want: fmt.Sprintf("%s - Ready: %s\n%s - Ready: %s\n%s - Ready: %s",
				master0Node.Name, corev1.ConditionTrue,
				master1Node.Name, corev1.ConditionFalse,
				master2Node.Name, corev1.ConditionUnknown),
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(master0Node)),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(master1Node)),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(master2Node)),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			kubernetescli := kubernetesfake.NewSimpleClientset(tt.objects...)

			h, log := testlog.New()

			m := &manager{
				log:           log,
				kubernetescli: kubernetescli,
			}

			got, gotErr := m.logNodes(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogClusterOperators(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		want     interface{}
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple CO output and logs full CO object",
			objects: []kruntime.Object{aroOperator, machineApiOperator},
			want: fmt.Sprintf("%s - Available: %s, Progressing: %s, Degraded: %s\n%s - Available: %s, Progressing: %s, Degraded: %s",
				aroOperator.Name, configv1.ConditionTrue, configv1.ConditionFalse, configv1.ConditionFalse,
				machineApiOperator.Name, configv1.ConditionFalse, configv1.ConditionUnknown, configv1.ConditionTrue),
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(aroOperator)),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(machineApiOperator)),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			configcli := configfake.NewSimpleClientset(tt.objects...)

			h, log := testlog.New()

			m := &manager{
				log:       log,
				configcli: configcli,
			}

			got, gotErr := m.logClusterOperators(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogIngressControllers(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		want     interface{}
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple IC output and logs full IC object",
			objects: []kruntime.Object{defaultIngressController},
			want: fmt.Sprintf("%s - Available: %s, Progressing: %s, Degraded: %s",
				defaultIngressController.Name, operatorv1.ConditionTrue, operatorv1.ConditionFalse, operatorv1.ConditionFalse),
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(asJson(defaultIngressController)),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			operatorcli := operatorfake.NewSimpleClientset(tt.objects...)

			h, log := testlog.New()

			m := &manager{
				log:         log,
				operatorcli: operatorcli,
			}

			got, gotErr := m.logIngressControllers(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogPodLogs(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		want     interface{}
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name: "no pods returns empty and logs nothing",
			want: []interface{}{},
		},
		{
			name:    "outputs status of aro-operator pods and directly logs pod logs",
			objects: []kruntime.Object{aroOperatorMasterPod, aroOperatorWorkerPod},
			want: []interface{}{
				fmt.Sprintf("pod status %s: %v", aroOperatorMasterPod.Name, aroOperatorMasterPod.Status),
				fmt.Sprintf("pod status %s: %v", aroOperatorWorkerPod.Name, aroOperatorWorkerPod.Status),
			},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("fake logs"),
					"pod":   gomega.Equal(aroOperatorMasterPod.Name),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("fake logs"),
					"pod":   gomega.Equal(aroOperatorWorkerPod.Name),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			kubernetescli := kubernetesfake.NewSimpleClientset(tt.objects...)

			h, log := testlog.New()

			m := &manager{
				log:           log,
				kubernetescli: kubernetescli,
			}

			got, gotErr := m.logPodLogs(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
			assert.Equal(t, tt.want, got)
		})
	}
}

func asJson(r kruntime.Object) string {
	r = r.DeepCopyObject()
	a, _ := meta.Accessor(r)
	a.SetManagedFields(nil)

	json, _ := json.Marshal(r)
	return string(json)
}
