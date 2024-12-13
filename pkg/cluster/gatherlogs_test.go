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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var (
	managedFields = []metav1.ManagedFieldsEntry{{Manager: "something"}}
	cvv           = &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version"}}
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
	aroOperator              = &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: "aro", ManagedFields: managedFields}}
	machineApiOperator       = &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: "machine-api", ManagedFields: managedFields}}
	defaultIngressController = &operatorv1.IngressController{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-ingress-operator", Name: "default", ManagedFields: managedFields}}
	aroOperatorMasterPod     = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-azure-operator", Name: "aro-operator-master-aaaaaaaaa-aaaaa"}, Status: corev1.PodStatus{}}
	aroOperatorWorkerPod     = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-azure-operator", Name: "aro-operator-worker-bbbbbbbbb-bbbbb"}, Status: corev1.PodStatus{}}
)

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
			want:    cvv,
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
			want: fmt.Sprintf(`%s Ready: %s
%s Ready: %s
%s Ready: %s`,
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
			kubernetescli := fake.NewSimpleClientset(tt.objects...)

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
		name    string
		objects []kruntime.Object
		want    interface{}
		wantErr string
	}{
		{
			name:    "returns COs without managed fields",
			objects: []kruntime.Object{aroOperator, machineApiOperator},
			want: []configv1.ClusterOperator{
				{ObjectMeta: metav1.ObjectMeta{Name: "aro"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "machine-api"}},
			},
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

			got, gotErr := m.logClusterOperators(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogIngressControllers(t *testing.T) {
	for _, tt := range []struct {
		name    string
		objects []kruntime.Object
		want    interface{}
		wantErr string
	}{
		{
			name:    "returns ICs without managed fields",
			objects: []kruntime.Object{defaultIngressController},
			want: []operatorv1.IngressController{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "openshift-ingress-operator", Name: "default"}},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			operatorcli := operatorfake.NewSimpleClientset(tt.objects...)

			_, log := testlog.New()

			m := &manager{
				log:         log,
				operatorcli: operatorcli,
			}

			got, gotErr := m.logIngressControllers(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
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
			kubernetescli := fake.NewSimpleClientset(tt.objects...)

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
