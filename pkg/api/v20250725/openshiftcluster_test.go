package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

func TestOpenShiftClusterUnmarshalJSONTags(t *testing.T) {
	value := "old"
	newValue := "value"
	tests := []struct {
		name string
		json string
		want map[string]*string
	}{
		{
			name: "present tags replace existing tags",
			json: `{"tags":{"new":"value"}}`,
			want: map[string]*string{"new": &newValue},
		},
		{
			name: "empty tags remove existing tags",
			json: `{"tags":{}}`,
			want: map[string]*string{},
		},
		{
			name: "null tags remove existing tags",
			json: `{"tags":null}`,
			want: nil,
		},
		{
			name: "omitted tags preserve existing tags",
			json: `{}`,
			want: map[string]*string{"old": &value},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			old := "old"
			cluster := &OpenShiftCluster{OpenShiftCluster: generated.OpenShiftCluster{
				Tags: map[string]*string{"old": &old},
			}}
			if err := json.Unmarshal([]byte(test.json), cluster); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cluster.Tags, test.want) {
				t.Fatalf("got tags %#v, want %#v", cluster.Tags, test.want)
			}
		})
	}
}

func TestIsWorkloadIdentity(t *testing.T) {
	tests := []*struct {
		name string
		oc   OpenShiftCluster
		want bool
	}{
		{
			name: "Cluster is Workload Identity",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         nil,
				}},
			},
			want: true,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         &generated.ServicePrincipalProfile{},
				}},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: nil,
					ServicePrincipalProfile:         nil,
				}},
			},
			want: false,
		},
		{
			name: "Cluster is Service Principal",
			oc: OpenShiftCluster{
				OpenShiftCluster: generated.OpenShiftCluster{Properties: &generated.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{},
					ServicePrincipalProfile:         &generated.ServicePrincipalProfile{},
				}},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.oc.UsesWorkloadIdentity()
			if got != test.want {
				t.Error(fmt.Errorf("got != want: %v != %v", got, test.want))
			}
		})
	}
}
