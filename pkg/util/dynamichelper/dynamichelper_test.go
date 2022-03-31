package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
	"time"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestMerge(t *testing.T) {
	serviceInternalTrafficPolicy := corev1.ServiceInternalTrafficPolicyCluster
	var terminateGracePeriod int64 = corev1.DefaultTerminationGracePeriodSeconds
	var revisionHistoryLimit int32 = 10
	var replicasDeployment int32 = 1
	var progressDlineSec int32 = 600

	for _, tt := range []struct {
		name          string
		old           kruntime.Object
		new           kruntime.Object
		want          kruntime.Object
		wantChanged   bool
		wantEmptyDiff bool
	}{
		{
			name: "general merge",
			old: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					SelfLink:          "selfLink",
					UID:               "uid",
					ResourceVersion:   "1",
					CreationTimestamp: metav1.Time{Time: time.Unix(0, 0)},
					Labels: map[string]string{
						"key": "value",
					},
					Annotations: map[string]string{
						"key":                     "value",
						"openshift.io/sa.scc.mcs": "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
				},
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{
						"finalizer",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			new: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"openshift.io/node-selector": "",
					},
				},
			},
			want: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					SelfLink:          "selfLink",
					UID:               "uid",
					ResourceVersion:   "1",
					CreationTimestamp: metav1.Time{Time: time.Unix(0, 0)},
					Annotations: map[string]string{
						"openshift.io/node-selector":              "",
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{"kubernetes.io/metadata.name": "test"},
				},
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{
						"finalizer",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			wantChanged: true,
		},
		{
			name: "Namespace no changes",
			old: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{
						"olm.operatorgroup.uid/jdfgbdfgdfhg": "test",
					},
				},
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{
						"finalizer",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			new: &corev1.Namespace{},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{
						"olm.operatorgroup.uid/jdfgbdfgdfhg": "test",
					},
				},
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{
						"finalizer",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "ServiceAccount no changes",
			old: &corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{
						Name: "secret1",
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "pullsecret1",
					},
				},
			},
			new: &corev1.ServiceAccount{},
			want: &corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{
						Name: "secret1",
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "pullsecret1",
					},
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "ConfigMap with injected ca bundle label and bundle, no changes",
			old: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{
					"ca-bundle.crt": "bundlehere",
				},
			},
			new: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{
					"ca-bundle.crt": "bundlehere",
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "ConfigMap with injected ca bundle label and no bundle, no changes",
			old: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{},
			},
			new: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{},
			},
			wantEmptyDiff: true,
		},
		{
			name: "Service no changes",
			old: &corev1.Service{
				Spec: corev1.ServiceSpec{
					ClusterIP:             "1.2.3.4",
					Type:                  corev1.ServiceTypeClusterIP,
					SessionAffinity:       corev1.ServiceAffinityNone,
					InternalTrafficPolicy: &serviceInternalTrafficPolicy,
				},
			},
			new: &corev1.Service{},
			want: &corev1.Service{
				Spec: corev1.ServiceSpec{
					ClusterIP:             "1.2.3.4",
					Type:                  corev1.ServiceTypeClusterIP,
					SessionAffinity:       corev1.ServiceAffinityNone,
					InternalTrafficPolicy: &serviceInternalTrafficPolicy,
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "DaemonSet changes",
			old: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"deprecated.daemonset.template.generation": "1",
					},
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 5,
					NumberReady:            5,
					ObservedGeneration:     1,
				},
			},
			new: &appsv1.DaemonSet{},
			want: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"deprecated.daemonset.template.generation": "1",
					},
				},
				Status: appsv1.DaemonSetStatus{
					CurrentNumberScheduled: 5,
					NumberReady:            5,
					ObservedGeneration:     1,
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Always",
							TerminationGracePeriodSeconds: &terminateGracePeriod,
							DNSPolicy:                     "ClusterFirst",
							SecurityContext:               &corev1.PodSecurityContext{},
							SchedulerName:                 "default-scheduler",
						},
					},
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
						Type: appsv1.RollingUpdateDaemonSetStrategyType,
						RollingUpdate: &appsv1.RollingUpdateDaemonSet{
							MaxUnavailable: &intstr.IntOrString{IntVal: 1},
							MaxSurge:       &intstr.IntOrString{IntVal: 0},
						},
					},
					RevisionHistoryLimit: &revisionHistoryLimit,
				},
			},
			wantChanged: true,
		},
		{
			name: "Deployment changes",
			old: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "2",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DeprecatedServiceAccount: "openshift-apiserver-sa",
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 3,
					ReadyReplicas:     3,
					Replicas:          3,
					UpdatedReplicas:   3,
				},
			},
			new: &appsv1.Deployment{},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "2",
					},
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 3,
					ReadyReplicas:     3,
					Replicas:          3,
					UpdatedReplicas:   3,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicasDeployment,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Always",
							TerminationGracePeriodSeconds: &terminateGracePeriod,
							DNSPolicy:                     "ClusterFirst",
							SecurityContext:               &corev1.PodSecurityContext{},
							SchedulerName:                 "default-scheduler",
							DeprecatedServiceAccount:      "openshift-apiserver-sa",
						},
					},
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &appsv1.RollingUpdateDeployment{
							MaxUnavailable: &intstr.IntOrString{
								Type:   1,
								StrVal: "25%",
							},
							MaxSurge: &intstr.IntOrString{
								Type:   1,
								StrVal: "25%",
							},
						},
					},
					RevisionHistoryLimit:    &revisionHistoryLimit,
					ProgressDeadlineSeconds: &progressDlineSec,
				},
			},
			wantChanged: true,
		},
		{
			name: "KubeletConfig no changes",
			old: &mcv1.KubeletConfig{
				Status: mcv1.KubeletConfigStatus{
					Conditions: []mcv1.KubeletConfigCondition{
						{
							Message: "Success",
							Status:  "True",
							Type:    "Success",
						},
					},
				},
			},
			new: &mcv1.KubeletConfig{},
			want: &mcv1.KubeletConfig{
				Status: mcv1.KubeletConfigStatus{
					Conditions: []mcv1.KubeletConfigCondition{
						{
							Message: "Success",
							Status:  "True",
							Type:    "Success",
						},
					},
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "Cluster no changes",
			old: &arov1alpha1.Cluster{
				Status: arov1alpha1.ClusterStatus{
					OperatorVersion: "8b66c40",
				},
			},
			new: &arov1alpha1.Cluster{},
			want: &arov1alpha1.Cluster{
				Status: arov1alpha1.ClusterStatus{
					OperatorVersion: "8b66c40",
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "CustomResourceDefinition Betav1 no changes",
			old: &extensionsv1beta1.CustomResourceDefinition{
				Status: extensionsv1beta1.CustomResourceDefinitionStatus{
					Conditions: []extensionsv1beta1.CustomResourceDefinitionCondition{
						{
							Message: "no conflicts found",
							Reason:  "NoConflicts",
						},
					},
				},
			},
			new: &extensionsv1beta1.CustomResourceDefinition{},
			want: &extensionsv1beta1.CustomResourceDefinition{
				Status: extensionsv1beta1.CustomResourceDefinitionStatus{
					Conditions: []extensionsv1beta1.CustomResourceDefinitionCondition{
						{
							Message: "no conflicts found",
							Reason:  "NoConflicts",
						},
					},
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "CustomResourceDefinition no changes",
			old: &extensionsv1.CustomResourceDefinition{
				Status: extensionsv1.CustomResourceDefinitionStatus{
					Conditions: []extensionsv1.CustomResourceDefinitionCondition{
						{
							Message: "no conflicts found",
							Reason:  "NoConflicts",
						},
					},
				},
			},
			new: &extensionsv1.CustomResourceDefinition{},
			want: &extensionsv1.CustomResourceDefinition{
				Spec: extensionsv1.CustomResourceDefinitionSpec{
					Conversion: &extensionsv1.CustomResourceConversion{
						Strategy: "None",
					},
				},
				Status: extensionsv1.CustomResourceDefinitionStatus{
					Conditions: []extensionsv1.CustomResourceDefinitionCondition{
						{
							Message: "no conflicts found",
							Reason:  "NoConflicts",
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "Secret changes, not logged",
			old: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("old"),
				},
			},
			new: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("new"),
				},
			},
			want: &corev1.Secret{
				Data: map[string][]byte{
					"secret": []byte("new"),
				},
				Type: corev1.SecretTypeOpaque,
			},
			wantChanged:   true,
			wantEmptyDiff: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, diff, err := merge(tt.old, tt.new)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
			if changed != tt.wantChanged {
				t.Error(changed)
			}
			if diff == "" != tt.wantEmptyDiff {
				t.Error(diff)
			}
		})
	}
}
