package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	hivev1 "github.com/openshift/hive/apis/hive/v1"

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
		name     string
		doc      *api.OpenShiftClusterDocument
		cd       *hivev1.ClusterDeployment
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name: "no clusterdoc returns empty",
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`skipping step`),
				},
			},
		},
		{
			name: "no clusterdeployment returns error",
			doc:  doc,
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal(`failed to get cluster deployment`),
					"error": gomega.MatchError(`clusterdeployments.hive.openshift.io "cluster" not found`),
				},
			},
			wantErr: `clusterdeployments.hive.openshift.io "cluster" not found`,
		},
		{
			name: "clusterdeployment present returns clusterdeployment without managed fields",
			doc:  doc,
			cd:   cd,
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusterdeployment cluster - {'metadata':{'name':'cluster','creationTimestamp':null},'spec':{'clusterName':'','baseDomain':'','platform':{},'controlPlaneConfig':{'servingCertificates':{}},'installed':false},'status':{}}`),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx := context.Background()
			h, log := testlog.New()

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

			gotErr := m.logClusterDeployment(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}

func TestLogClusterVersion(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "no cv resource returns err",
			wantErr: `clusterversions.config.openshift.io "version" not found`,
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal(`failed to get clusterversion`),
					"error": gomega.MatchError(`clusterversions.config.openshift.io "version" not found`),
				},
			},
		},
		{
			name:    "returns cv resource if present",
			objects: []kruntime.Object{cvv},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusterversion version - {'metadata':{'name':'version','creationTimestamp':null},'spec':{'clusterID':''},'status':{'desired':{'version':'','image':''},'observedGeneration':0,'versionHash':'','capabilities':{},'availableUpdates':null}}`),
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

			gotErr := m.logClusterVersion(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}

func TestLogNodes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		want     []string
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple node output and logs full node object",
			objects: []kruntime.Object{master0Node, master1Node, master2Node},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-0 - Ready: True`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-0 - {'metadata':{'name':'cluster-aaaaa-master-0','creationTimestamp':null},'spec':{},'status':{'conditions':[{'type':'Ready','status':'True','lastHeartbeatTime':null,'lastTransitionTime':null}],'daemonEndpoints':{'kubeletEndpoint':{'Port':0}},'nodeInfo':{'machineID':'','systemUUID':'','bootID':'','kernelVersion':'','osImage':'','containerRuntimeVersion':'','kubeletVersion':'','kubeProxyVersion':'','operatingSystem':'','architecture':''}}}`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-1 - Ready: False`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-1 - {'metadata':{'name':'cluster-aaaaa-master-1','creationTimestamp':null},'spec':{},'status':{'conditions':[{'type':'Ready','status':'False','lastHeartbeatTime':null,'lastTransitionTime':null}],'daemonEndpoints':{'kubeletEndpoint':{'Port':0}},'nodeInfo':{'machineID':'','systemUUID':'','bootID':'','kernelVersion':'','osImage':'','containerRuntimeVersion':'','kubeletVersion':'','kubeProxyVersion':'','operatingSystem':'','architecture':''}}}`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-2 - Ready: Unknown`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`node cluster-aaaaa-master-2 - {'metadata':{'name':'cluster-aaaaa-master-2','creationTimestamp':null},'spec':{},'status':{'conditions':[{'type':'Ready','status':'Unknown','lastHeartbeatTime':null,'lastTransitionTime':null}],'daemonEndpoints':{'kubeletEndpoint':{'Port':0}},'nodeInfo':{'machineID':'','systemUUID':'','bootID':'','kernelVersion':'','osImage':'','containerRuntimeVersion':'','kubeletVersion':'','kubeProxyVersion':'','operatingSystem':'','architecture':''}}}`),
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

			gotErr := m.logNodes(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}

func TestLogClusterOperators(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple CO output and logs full CO object",
			objects: []kruntime.Object{aroOperator, machineApiOperator},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusteroperator aro - Available: True, Progressing: False, Degraded: False`)},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusteroperator aro - {'metadata':{'name':'aro','creationTimestamp':null},'spec':{},'status':{'conditions':[{'type':'Available','status':'True','lastTransitionTime':null},{'type':'Progressing','status':'False','lastTransitionTime':null},{'type':'Degraded','status':'False','lastTransitionTime':null}],'extension':null}}`)},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusteroperator machine-api - Available: False, Progressing: Unknown, Degraded: True`)},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`clusteroperator machine-api - {'metadata':{'name':'machine-api','creationTimestamp':null},'spec':{},'status':{'conditions':[{'type':'Available','status':'False','lastTransitionTime':null},{'type':'Progressing','status':'Unknown','lastTransitionTime':null},{'type':'Degraded','status':'True','lastTransitionTime':null}],'extension':null}}`)},
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

			gotErr := m.logClusterOperators(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}

func TestLogIngressControllers(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name:    "returns simple IC output and logs full IC object",
			objects: []kruntime.Object{defaultIngressController},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`ingresscontroller default - Available: True, Progressing: False, Degraded: False`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`ingresscontroller default - {'metadata':{'name':'default','namespace':'openshift-ingress-operator','creationTimestamp':null},'spec':{'httpErrorCodePages':{'name':''},'clientTLS':{'clientCertificatePolicy':'','clientCA':{'name':''}},'tuningOptions':{'reloadInterval':'0s'},'unsupportedConfigOverrides':null,'httpCompression':{}},'status':{'availableReplicas':0,'selector':'','domain':'','conditions':[{'type':'Available','status':'True','lastTransitionTime':null},{'type':'Progressing','status':'False','lastTransitionTime':null},{'type':'Degraded','status':'False','lastTransitionTime':null}]}}`),
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

			gotErr := m.logIngressControllers(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}

func TestLogPodLogs(t *testing.T) {
	for _, tt := range []struct {
		name     string
		objects  []kruntime.Object
		wantLogs []map[string]types.GomegaMatcher
		wantErr  string
	}{
		{
			name: "no pods returns empty and logs nothing",
		},
		{
			name:    "outputs status of aro-operator pods and directly logs pod logs",
			objects: []kruntime.Object{aroOperatorMasterPod, aroOperatorWorkerPod},
			wantLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pod openshift-azure-operator/aro-operator-master-aaaaaaaaa-aaaaa - phase= reason= message=`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`fake logs`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`pod openshift-azure-operator/aro-operator-worker-bbbbbbbbb-bbbbb - phase= reason= message=`),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`fake logs`),
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

			gotErr := m.logPodLogs(ctx)
			utilerror.AssertErrorMessage(t, gotErr, tt.wantErr)
			require.NoError(t, testlog.AssertLoggingOutput(h, tt.wantLogs))
		})
	}
}
