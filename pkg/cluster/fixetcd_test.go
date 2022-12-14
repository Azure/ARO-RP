package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
	fakekubecli "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	testEtcds   = "cluster"
	testCluster = testEtcds
	testInfraID = "-zfsbk-"
	nodePostfix = testInfraID + "master-2"
)

func TestFixEtcd(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr string
		pods    *corev1.PodList
		status  int
		scc     *securityv1.SecurityContextConstraints
		cluster string
	}{
		{
			name:    "pass: find degraded member",
			pods:    degradedPods(testCluster),
			cluster: testCluster,
		},
		{
			name:    "fail: get peer pods",
			wantErr: "degradedEtcd is empty, unable to remediate etcd deployment",
			pods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						TypeMeta:   metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{Name: "etcd-cluster-lfm4j-master-2"},
						Spec:       corev1.PodSpec{NodeName: "test-cluster-master-2"},
					},
				},
			},
		},
	} {
		ctx := context.WithValue(context.Background(), ctxKey, "TRUE")
		kubecli, err := newFakeKubecli(ctx, tt.pods, tt.cluster)
		if err != nil {
			t.Fatal(err)
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.String() {
			case "/apis/security.openshift.io/v1/securitycontextconstraints":
				buf := &bytes.Buffer{}
				err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(tt.scc)
				if err != nil {
					t.Log(err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("x-ms-version", "2018-12-31")
				_, err = w.Write(buf.Bytes())
				if err != nil {
					t.Fatal(err)
				}
			default:
				t.Logf("resource requested %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		securitycli := securityclient.NewForConfigOrDie(&rest.Config{
			Host: ts.URL,
			Impersonate: rest.ImpersonationConfig{
				UserName: "privileged",
				Groups:   []string{"*"},
			},
			ContentConfig: rest.ContentConfig{
				AcceptContentTypes: kruntime.ContentTypeJSON,
				GroupVersion: &kschema.GroupVersion{
					Group:   "",
					Version: "v1",
				},
				ContentType: kruntime.ContentTypeJSON,
			},
		})
		_, err = securitycli.SecurityV1().SecurityContextConstraints().Create(ctx, tt.scc, metav1.CreateOptions{})

		m := &manager{
			log: logrus.NewEntry(logrus.StandardLogger()),
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Name: testCluster,
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg",
						},
						InfraID: testInfraID,
					},
				},
			},
			securitycli:   securitycli,
			kubernetescli: kubecli,
			operatorcli:   operatorfake.NewSimpleClientset(newEtcds()),
		}
		wr := ktesting.DefaultWatchReactor(kubecli.InvokesWatch(ktesting.NewWatchAction(kschema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "Etcd",
		}, nameSpaceEtcds, metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Etcd",
			},
			LabelSelector:  jobNameDataBackup,
			Watch:          true,
			TimeoutSeconds: to.Int64Ptr(60),
		})))
		kubecli.AddWatchReactor(jobNameDataBackup, wr)

		kruntime.NewSchemeBuilder(func(*kruntime.Scheme) error {
			s := kruntime.NewScheme()
			s.AddKnownTypeWithName(kschema.GroupVersionKind{
				Kind:    "SecurityContextConstraints",
				Version: "v1",
				Group:   "security.openshift.io",
			}, tt.scc)
			return nil
		})

		t.Run(tt.name, func(t *testing.T) {
			err = m.fixEtcd(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%v\n !=\n%s", err, tt.wantErr))
			}
		})
	}
}

func newEtcds() kruntime.Object {
	return &operatorv1.Etcd{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: testEtcds,
		},
		Spec: operatorv1.EtcdSpec{},
	}
}

func newFakeKubecli(ctx context.Context, pods *corev1.PodList, cluster string) (*fakekubecli.Clientset, error) {
	p, err := pods.Marshal()
	if err != nil {
		return nil, err
	}

	secrets := &corev1.SecretList{
		TypeMeta: metav1.TypeMeta{},
		Items: []corev1.Secret{
			{
				TypeMeta: metav1.TypeMeta{Kind: "Etcd"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-peer-" + cluster + nodePostfix,
					Namespace: nameSpaceEtcds,
				},
				Type: corev1.SecretTypeTLS,
			},
			{
				TypeMeta: metav1.TypeMeta{Kind: "Etcd"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-serving-" + cluster + nodePostfix,
					Namespace: nameSpaceEtcds,
				},
				Type: corev1.SecretTypeTLS,
			},
			{
				TypeMeta: metav1.TypeMeta{Kind: "Etcd"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd-serving-metrics-" + cluster + nodePostfix,
					Namespace: nameSpaceEtcds,
				},
				Type: corev1.SecretTypeTLS,
			},
		},
	}
	etcds := &operatorv1.Etcd{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Etcd",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        testEtcds,
			Namespace:   nameSpaceEtcds,
			ClusterName: cluster,
		},
		Spec: operatorv1.EtcdSpec{
			StaticPodOperatorSpec: operatorv1.StaticPodOperatorSpec{
				OperatorSpec: operatorv1.OperatorSpec{
					ManagementState: operatorv1.ManagementState("Managed"),
					ObservedConfig: kruntime.RawExtension{
						Raw:    p,
						Object: pods,
					},
				},
			},
		},
	}

	kruntime.NewSchemeBuilder(func(*kruntime.Scheme) error {
		s := kruntime.NewScheme()
		s.AddKnownTypeWithName(kschema.GroupVersionKind{
			Kind:    "Etcd",
			Version: "v1",
		}, etcds)
		s.AddKnownTypes(kschema.GroupVersion{
			Group:   "",
			Version: "v1",
		}, secrets)
		return nil
	})

	kubecli := fakekubecli.NewSimpleClientset(secrets)
	for _, p := range pods.Items {
		kubecli.CoreV1().Pods(nameSpaceEtcds).Create(ctx, p.DeepCopy(), metav1.CreateOptions{})
	}

	return kubecli, nil
}

func degradedPods(cluster string) *corev1.PodList {
	nodeName := "etcd-" + cluster + nodePostfix
	return &corev1.PodList{
		TypeMeta: metav1.TypeMeta{},
		Items: []corev1.Pod{
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: nameSpaceEtcds,
				},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.4",
						},
						{
							IP: "127.0.0.2",
						},
						{
							IP: "127.0.0.1",
						},
					},
				},
				Spec: corev1.PodSpec{
					NodeName: nodeName,
					Containers: []corev1.Container{
						{
							Name: "etcd",
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_" + cluster + "_zfsbk_master_0_IP",
									Value: "127.0.0.1",
								},
								{
									Name:  "NODE_ " + cluster + "_zfsbk_master_1_IP",
									Value: "127.0.0.2",
								},
								{
									Name:  "NODE_" + cluster + "_zfsbk_master_2_IP",
									Value: "127.0.0.3",
								},
							},
						},
					},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "etcd",
					Namespace: nameSpaceEtcds,
				},
				Spec: corev1.PodSpec{
					NodeName: "master-42",
					Containers: []corev1.Container{
						{
							Name: "etcd",
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_" + cluster + "_master_0_IP",
									Value: "127.0.0.1",
								},
								{
									Name:  "NODE_ " + cluster + "_master_1_IP",
									Value: "127.0.0.2",
								},
								{
									Name:  "NODE_" + cluster + "_master_2_IP",
									Value: "127.0.0.3",
								},
							},
						},
					},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "etcd",
				},
				Spec: corev1.PodSpec{
					NodeName: "master-3",
					Containers: []corev1.Container{
						{
							Name: "etcd",
						},
					},
				},
			},
		},
	}
}
