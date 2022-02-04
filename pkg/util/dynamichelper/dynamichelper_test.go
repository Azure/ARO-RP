package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestMerge(t *testing.T) {
	serviceInternalTrafficPolicy := corev1.ServiceInternalTrafficPolicyCluster
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
