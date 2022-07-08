package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

type getUserDataSecretReferenceTestData struct {
	name        string
	objectMeta  *metav1.ObjectMeta
	machineSpec *machinev1beta1.MachineSpec
	result      *corev1.SecretReference
	shouldFail  bool
}

func marshalAzureMachineProviderSpec(t *testing.T, spec *machinev1beta1.AzureMachineProviderSpec) []byte {
	serializer := kjson.NewSerializerWithOptions(
		kjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme,
		kjson.SerializerOptions{Yaml: true},
	)

	yaml := scheme.Codecs.CodecForVersions(serializer, nil, schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()), nil)

	buf := &bytes.Buffer{}
	err := yaml.Encode(spec, buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func marshal(t *testing.T, i interface{}) []byte {
	b, err := json.Marshal(i)
	if err != nil {
		t.Fatal(i)
	}
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func userDataSecret(t *testing.T, namespace, name, appendSource, mergeSource string) *corev1.Secret {
	config := map[string]interface{}{
		"extrakey": true,
	}

	if appendSource != "" {
		config["append"] = []interface{}{
			map[string]interface{}{
				"extrakey": []interface{}{},
				"source":   appendSource,
			},
		}
	}

	if mergeSource != "" {
		config["merge"] = []interface{}{
			map[string]interface{}{
				"extrakey": map[string]interface{}{},
				"source":   appendSource,
			},
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"userData": marshal(t, map[string]interface{}{
				"extrakey": 1,
				"ignition": map[string]interface{}{
					"extrakey": "2",
					"config":   config,
				},
			}),
		},
	}
}

func testMachine(t *testing.T, namespace, name string, spec *machinev1beta1.AzureMachineProviderSpec) *machinev1beta1.Machine {
	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{
					Raw: marshalAzureMachineProviderSpec(t, spec),
				},
			},
		},
	}
}

func testMachineSet(t *testing.T, namespace, name string, spec *machinev1beta1.AzureMachineProviderSpec) *machinev1beta1.MachineSet {
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: machinev1beta1.MachineSetSpec{
			Template: machinev1beta1.MachineTemplateSpec{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: marshalAzureMachineProviderSpec(t, spec),
						},
					},
				},
			},
		},
	}
}

func TestFixMCSUserData(t *testing.T) {
	ctx := context.Background()

	m := &manager{
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Domain: "example.com",
					},
					APIServerProfile: api.APIServerProfile{
						IntIP: "1.2.3.4",
					},
				},
			},
		},
		kubernetescli: fake.NewSimpleClientset(
			userDataSecret(t, "openshift-machine-api", "master-user-data", "https://api-int.example.com:22623/config/master", ""),
			userDataSecret(t, "openshift-machine-api", "worker-user-data", "", "https://api-int.example.com:22623/config/worker"),
		),
		maocli: machinefake.NewSimpleClientset(
			testMachineSet(t, "openshift-machine-api", "worker", &machinev1beta1.AzureMachineProviderSpec{
				UserDataSecret: &corev1.SecretReference{
					Name: "worker-user-data",
				},
			}),
			testMachine(t, "openshift-machine-api", "master", &machinev1beta1.AzureMachineProviderSpec{
				UserDataSecret: &corev1.SecretReference{
					Name: "master-user-data",
				},
			}),
		),
	}

	wantSecrets := []*corev1.Secret{
		userDataSecret(t, "openshift-machine-api", "master-user-data", "https://1.2.3.4:22623/config/master", ""),
		userDataSecret(t, "openshift-machine-api", "worker-user-data", "", "https://1.2.3.4:22623/config/worker"),
	}

	err := m.fixMCSUserData(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, wantSecret := range wantSecrets {
		s, err := m.kubernetescli.CoreV1().Secrets(wantSecret.Namespace).Get(ctx, wantSecret.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(s, wantSecret) {
			t.Error(cmp.Diff(s, wantSecret))
		}
	}
}

func TestGetUserDataSecretReference(t *testing.T) {
	for _, td := range []getUserDataSecretReferenceTestData{
		{
			name:       "valid cluster-api-provider-azure spec",
			objectMeta: &metav1.ObjectMeta{Namespace: "any"},
			machineSpec: &machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &kruntime.RawExtension{
						Raw: []byte(`{
								"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
								"kind": "AzureMachineProviderSpec",
								"userDataSecret": {"name": "any"}
							}`),
					},
				},
			},
			result: &corev1.SecretReference{
				Name:      "any",
				Namespace: "any",
			},
			shouldFail: false,
		},
		{
			name:       "valid openshift/api spec",
			objectMeta: &metav1.ObjectMeta{Namespace: "another"},
			machineSpec: &machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &kruntime.RawExtension{
						Raw: []byte(`{
								"apiVersion": "machine.openshift.io/v1beta1",
								"kind": "AzureMachineProviderSpec",
								"userDataSecret": {"name": "any"}
							}`),
					},
				},
			},
			result: &corev1.SecretReference{
				Name:      "any",
				Namespace: "another",
			},
			shouldFail: false,
		},
		{
			name:       "not valid spec",
			objectMeta: &metav1.ObjectMeta{Namespace: "any"},
			machineSpec: &machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &kruntime.RawExtension{
						Raw: []byte(`{
								"apiVersion": "apiversion.openshift.io/unknown",
								"kind": "AzureMachineProviderSpec"
							}`),
					},
				},
			},
			shouldFail: true,
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			res, err := getUserDataSecretReference(td.objectMeta, td.machineSpec)
			if err != nil {
				if !td.shouldFail {
					t.Errorf("error hasn't been expected: %v", err)
				}
				return
			}
			if !reflect.DeepEqual(res, td.result) {
				t.Errorf("unexpected result: %+v", res)
			}
		})
	}
}
