package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/azure"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
)

func TestEnsureDeleted(t *testing.T) {
	ctx := context.Background()

	builder := fake.NewClientBuilder().WithRuntimeObjects(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name-3",
			Namespace: "test-ns-3",
		},
	})

	client := testclienthelper.NewHookingClient(builder.Build()).
		WithPreDeleteHook(func(obj client.Object) error {
			if obj.GetName() == "test-name-2" {
				return fmt.Errorf("error on %s", obj.GetName())
			}
			return nil
		})

	ch := NewWithClient(logrus.NewEntry(logrus.StandardLogger()), client)

	err := ch.EnsureDeleted(ctx,
		schema.GroupVersionKind{Group: "core", Version: "v1", Kind: "ConfigMap"},
		types.NamespacedName{
			Name:      "test-name-1",
			Namespace: "test-ns-1",
		})
	if err != nil {
		t.Errorf("no error should be bounced for status not found, but got: %v", err)
	}

	err = ch.EnsureDeleted(ctx, schema.GroupVersionKind{Group: "core", Version: "v1", Kind: "ConfigMap"},
		types.NamespacedName{
			Name:      "test-name-2",
			Namespace: "test-ns-2",
		})
	if err == nil {
		t.Error(fmt.Errorf("function should handle failure response (non-404) correctly: %w", err))
	}

	err = ch.EnsureDeleted(ctx, schema.GroupVersionKind{Group: "core", Version: "v1", Kind: "ConfigMap"},
		types.NamespacedName{
			Name:      "test-name-3",
			Namespace: "test-ns-3",
		})
	if err != nil {
		t.Errorf("function should handle success response correctly")
	}
}

func TestMerge(t *testing.T) {
	serviceInternalTrafficPolicy := corev1.ServiceInternalTrafficPolicyCluster

	mhc := &machinev1beta1.MachineHealthCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-machinehealthcheck",
			Namespace: "openshift-machine-api",
		},
		Spec: machinev1beta1.MachineHealthCheckSpec{
			Selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "machine.openshift.io/cluster-api-machineset",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
			},
			UnhealthyConditions: []machinev1beta1.UnhealthyCondition{
				{
					Type:    corev1.NodeReady,
					Timeout: metav1.Duration{Duration: 15 * time.Minute},
					Status:  corev1.ConditionFalse,
				},
			},
			NodeStartupTimeout: &metav1.Duration{Duration: 25 * time.Minute},
		},
	}

	mhcWithStatus := mhc.DeepCopy()
	mhcWithStatus.Status = machinev1beta1.MachineHealthCheckStatus{
		Conditions: machinev1beta1.Conditions{
			{
				Type:               machinev1beta1.RemediationAllowedCondition,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now()},
			},
		},
		CurrentHealthy:      to.IntPtr(3),
		ExpectedMachines:    to.IntPtr(3),
		RemediationsAllowed: 1,
	}

	mhcWithAnnotation := mhc.DeepCopy()
	mhcWithAnnotation.ObjectMeta.Annotations = map[string]string{
		"cluster.x-k8s.io/paused": "",
	}

	mhcWithStatusAndAnnotation := mhc.DeepCopy()
	mhcWithStatusAndAnnotation.Status = *mhcWithStatus.Status.DeepCopy()
	mhcWithStatusAndAnnotation.ObjectMeta.Annotations = mhcWithAnnotation.ObjectMeta.Annotations

	for _, tt := range []struct {
		name             string
		old              client.Object
		new              client.Object
		want             client.Object
		wantChanged      bool
		wantEmptyDiff    bool
		wantScrubbedDiff bool
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"openshift.io/internal-registry-pull-secret-ref": "example",
					},
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"openshift.io/internal-registry-pull-secret-ref": "example",
					},
				},
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
					Annotations: map[string]string{
						"openshift.io/owning-component": "Some Component",
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
					Annotations: map[string]string{
						"openshift.io/owning-component": "Some Component",
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
							TerminationGracePeriodSeconds: to.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
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
					RevisionHistoryLimit: to.Int32Ptr(10),
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
					Replicas: to.Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Always",
							TerminationGracePeriodSeconds: to.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
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
					RevisionHistoryLimit:    to.Int32Ptr(10),
					ProgressDeadlineSeconds: to.Int32Ptr(600),
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
			name: "CustomResourceDefinition changes",
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
			name: "ValidatingWebhookConfiguration changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Fail"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "ValidatingWebhookConfiguration no changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged:   false,
			wantEmptyDiff: true,
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
			wantChanged:      true,
			wantScrubbedDiff: true,
		},
		{
			name: "Hive ClusterDeployment no changes",
			old: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"hive.openshift.io/version":      "4.13.11",
						"hive.openshift.io/somemetadata": "bar",
					},
					Finalizers: []string{"bar"},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{
						Platform: &hivev1.ClusterPlatformMetadata{
							Azure: &azure.Metadata{
								ResourceGroupName: pointerutils.ToPtr("test"),
							},
						},
					},
				},
				Status: hivev1.ClusterDeploymentStatus{
					APIURL: "example",
				},
			},
			new: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"hive.openshift.io/somemetadata": "baz",
					},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{},
				},
			},
			want: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"hive.openshift.io/version":      "4.13.11",
						"hive.openshift.io/somemetadata": "bar",
					},
					Finalizers: []string{"bar"},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{
						Platform: &hivev1.ClusterPlatformMetadata{
							Azure: &azure.Metadata{
								ResourceGroupName: pointerutils.ToPtr("test"),
							},
						},
					},
				},
				Status: hivev1.ClusterDeploymentStatus{
					APIURL: "example",
				},
			},
			wantChanged:   false,
			wantEmptyDiff: true,
		},
		{
			name: "Hive ClusterDeployment changes",
			old: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"hive.openshift.io/version": "4.13.11",
					},
					Annotations: map[string]string{
						"hive.openshift.io/additional-log-fields": "oldlog",
					},
					Finalizers: []string{"bar"},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{
						Platform: &hivev1.ClusterPlatformMetadata{
							Azure: &azure.Metadata{
								ResourceGroupName: pointerutils.ToPtr("test"),
							},
						},
					},
				},
				Status: hivev1.ClusterDeploymentStatus{
					APIURL: "example",
				},
			},
			new: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
					Annotations: map[string]string{
						"hive.openshift.io/additional-log-fields": "log log log",
					},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{},
				},
			},
			want: &hivev1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"hive.openshift.io/version": "4.13.11",
					},
					Annotations: map[string]string{
						"hive.openshift.io/additional-log-fields": "log log log",
					},
					Finalizers: []string{"bar"},
				},
				Spec: hivev1.ClusterDeploymentSpec{
					ClusterMetadata: &hivev1.ClusterMetadata{
						Platform: &hivev1.ClusterPlatformMetadata{
							Azure: &azure.Metadata{
								ResourceGroupName: pointerutils.ToPtr("test"),
							},
						},
					},
				},
				Status: hivev1.ClusterDeploymentStatus{
					APIURL: "example",
				},
			},
			wantChanged: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, diff, err := Merge(tt.old, tt.new)
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
			if diff == "<scrubbed>" != tt.wantScrubbedDiff {
				t.Error(diff)
			}
		})
	}
}

func getFailurePolicyType(in string) *admissionregistrationv1.FailurePolicyType {
	fail := admissionregistrationv1.FailurePolicyType(in)
	return &fail
}

func TestMergeApply(t *testing.T) {
	serviceInternalTrafficPolicy := corev1.ServiceInternalTrafficPolicyCluster

	for _, tt := range []struct {
		name          string
		old           client.Object
		new           client.Object
		want          client.Object
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
					Name: "testobj",
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
			new: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
				},
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
					Annotations: map[string]string{
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{
						"olm.operatorgroup.uid/jdfgbdfgdfhg": "test",
						"kubernetes.io/metadata.name":        "testobj",
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
			wantChanged:   true,
		},
		{
			name: "Namespace no changes",
			old: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
					Annotations: map[string]string{
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{
						"olm.operatorgroup.uid/jdfgbdfgdfhg": "test",
						"kubernetes.io/metadata.name":        "testobj",
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
					Name: "testobj",
				},
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
					Annotations: map[string]string{
						"openshift.io/sa.scc.mcs":                 "mcs",
						"openshift.io/sa.scc.supplemental-groups": "groups",
						"openshift.io/sa.scc.uid-range":           "uids",
					},
					Labels: map[string]string{
						"olm.operatorgroup.uid/jdfgbdfgdfhg": "test",
						"kubernetes.io/metadata.name":        "testobj",
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
			wantChanged:   false,
		},
		{
			name: "ServiceAccount no changes",
			old: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
			new: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
			},
			want: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
					Name:      "testobj",
					Namespace: "testnamespace",
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
					Name:      "testobj",
					Namespace: "testnamespace",
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
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
					Name:      "testobj",
					Namespace: "testnamespace",
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{},
			},
			new: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: map[string]string{},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
					Labels: map[string]string{
						"config.openshift.io/inject-trusted-cabundle": "",
					},
				},
				Data: nil,
			},
			wantEmptyDiff: true,
		},
		{
			name: "Service no changes",
			old: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP:             "1.2.3.4",
					Type:                  corev1.ServiceTypeClusterIP,
					SessionAffinity:       corev1.ServiceAffinityNone,
					InternalTrafficPolicy: &serviceInternalTrafficPolicy,
				},
			},
			new: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
					Name:      "testobj",
					Namespace: "testnamespace",
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
			new: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: appsv1.DaemonSetSpec{
					RevisionHistoryLimit: to.Int32Ptr(12),
				},
			},
			want: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
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
							TerminationGracePeriodSeconds: to.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
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
					RevisionHistoryLimit: to.Int32Ptr(12),
				},
			},
			wantChanged: true,
		},
		{
			name: "Deployment changes",
			old: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
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
			new: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: appsv1.DeploymentSpec{
					RevisionHistoryLimit: to.Int32Ptr(12),
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
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
					Replicas: to.Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy:                 "Always",
							TerminationGracePeriodSeconds: to.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
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
					RevisionHistoryLimit:    to.Int32Ptr(12),
					ProgressDeadlineSeconds: to.Int32Ptr(600),
				},
			},
			wantChanged: true,
		},
		{
			name: "KubeletConfig no changes",
			old: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
			new: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
			},
			want: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
				},
				Status: arov1alpha1.ClusterStatus{
					OperatorVersion: "8b66c40",
				},
			},
			new: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
				},
			},
			want: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testobj",
				},
				Status: arov1alpha1.ClusterStatus{
					OperatorVersion: "8b66c40",
				},
			},
			wantEmptyDiff: true,
		},
		{
			name: "CustomResourceDefinition no changes",
			old: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: extensionsv1.CustomResourceDefinitionSpec{
					Conversion: &extensionsv1.CustomResourceConversion{
						Strategy: extensionsv1.WebhookConverter,
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
			new: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: extensionsv1.CustomResourceDefinitionSpec{
					Conversion: &extensionsv1.CustomResourceConversion{
						Strategy: extensionsv1.WebhookConverter,
					},
				},
			},
			want: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: extensionsv1.CustomResourceDefinitionSpec{
					Conversion: &extensionsv1.CustomResourceConversion{
						Strategy: extensionsv1.WebhookConverter,
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
			wantChanged: false,
		},
		{
			name: "CustomResourceDefinition changes",
			old: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Spec: extensionsv1.CustomResourceDefinitionSpec{
					Conversion: &extensionsv1.CustomResourceConversion{
						Strategy: extensionsv1.WebhookConverter,
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
			new: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
			},
			want: &extensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
			name: "ValidatingWebhookConfiguration changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Fail"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "ValidatingWebhookConfiguration no changes",
			old: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			new: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			want: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						FailurePolicy: getFailurePolicyType("Ignore"),
					},
				},
			},
			wantChanged:   false,
			wantEmptyDiff: true,
		},
		{
			name: "Secret changes, not logged",
			old: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "testobj",
					Namespace:       "testnamespace",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					"secret": []byte("old"),
				},
			},
			new: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
				Data: map[string][]byte{
					"secret": []byte("new"),
				},
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testobj",
					Namespace: "testnamespace",
				},
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
			ctx := context.Background()

			gvks, _, err := scheme.Scheme.ObjectKinds(tt.old)
			if err != nil {
				t.Error(err)
			}

			tt.old.GetObjectKind().SetGroupVersionKind(gvks[0])
			tt.new.GetObjectKind().SetGroupVersionKind(gvks[0])
			tt.want.GetObjectKind().SetGroupVersionKind(gvks[0])

			beenChanged := false
			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.old)

			clientFake := testclienthelper.NewHookingClient(builder.Build()).
				WithPostUpdateHook(func(obj client.Object) error {
					beenChanged = true
					return nil
				})

			ch := &clientHelper{
				Client: clientFake,
				log:    logrus.NewEntry(logrus.StandardLogger()),
			}

			err = ch.ensureOne(ctx, tt.new)
			if err != nil {
				t.Error(err)
			}

			got, err := scheme.Scheme.New(tt.old.GetObjectKind().GroupVersionKind())
			if err != nil {
				t.Error(err)
			}
			gotObj := got.(client.Object)

			err = clientFake.Get(ctx, client.ObjectKeyFromObject(tt.old), gotObj)
			if err != nil {
				t.Error(err)
			}

			// Don't test for the resourceversion
			gotObj.SetResourceVersion("")

			if !reflect.DeepEqual(got, tt.want) {
				for _, r := range deep.Equal(got, tt.want) {
					t.Error(r)
				}
			}
			if beenChanged != tt.wantChanged {
				t.Errorf("changed: %t, want: %t", beenChanged, tt.wantChanged)
			}
		})
	}
}

func TestGetOne(t *testing.T) {
	for _, tt := range []struct {
		name     string
		query    types.NamespacedName
		existing []runtime.Object
		want     client.Object
		wantErr  error
	}{
		{
			name:  "fetch success",
			query: types.NamespacedName{Name: "funobj", Namespace: "somewhere"},
			existing: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "funobj",
						Namespace:       "somewhere",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					StringData: map[string]string{
						"secret": "squirrels",
					},
					Type: corev1.SecretTypeOpaque,
				},
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "funobj",
					Namespace:       "somewhere",
					ResourceVersion: "1",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				StringData: map[string]string{
					"secret": "squirrels",
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name:     "fetch failure",
			query:    types.NamespacedName{Name: "funobj", Namespace: "somewhere"},
			existing: []runtime.Object{},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "funobj",
					Namespace:       "somewhere",
					ResourceVersion: "1",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				StringData: map[string]string{
					"secret": "squirrels",
				},
				Type: corev1.SecretTypeOpaque,
			},
			wantErr: &errors.StatusError{ErrStatus: metav1.Status{Message: `secrets "funobj" not found`}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			clientFake := fake.NewClientBuilder().WithRuntimeObjects(tt.existing...).Build()
			ch := NewWithClient(logrus.NewEntry(logrus.StandardLogger()), clientFake)

			gvks, _, err := scheme.Scheme.ObjectKinds(tt.want)
			if err != nil {
				t.Error(err)
			}

			c, err := scheme.Scheme.New(gvks[0])
			if err != nil {
				t.Error(err)
			}

			err = ch.GetOne(ctx, tt.query, c)
			if tt.wantErr == nil && err != nil {
				t.Fatal(err)
			} else if tt.wantErr != nil {
				for _, r := range deep.Equal(tt.wantErr, err) {
					t.Error(r)
				}
				return
			}

			if !reflect.DeepEqual(c, tt.want) {
				for _, r := range deep.Equal(c, tt.want) {
					t.Error(r)
				}
			}
		})
	}
}
