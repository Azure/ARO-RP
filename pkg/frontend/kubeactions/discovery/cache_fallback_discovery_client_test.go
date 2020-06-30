package discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_discovery "github.com/Azure/ARO-RP/pkg/util/mocks/kube"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

// TestVersion makes sure that bindata contains cache generated
// with the supported OpenShift version.
// To update discovery cache:
//   1. Create a new cluster
//	 2. Run `oc login` against this cluster or set KUBECONFIG env variable
//   3. Run `go run ./hack/gendiscoverycache/gendiscoverycache.go`
//   4. Run `make generate`
func TestVersion(t *testing.T) {
	b, err := Asset("assets_version")
	if err != nil {
		t.Fatal(err)
	}

	assetsVersion := string(b)
	if assetsVersion != version.OpenShiftVersion {
		t.Error(assetsVersion)
	}
}

func TestCacheFallbackDiscoveryClient(t *testing.T) {
	log := utillog.GetLogger()

	fakeServerGroups := &metav1.APIGroupList{
		Groups: []metav1.APIGroup{
			{
				Name: "foo",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "foo/v1",
						Version:      "v1",
					},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: "foo/v1",
					Version:      "v1",
				},
			},
		},
	}
	wantGroups := []*metav1.APIGroup{}
	for i := range fakeServerGroups.Groups {
		wantGroups = append(wantGroups, &fakeServerGroups.Groups[i])
	}

	fakeServerResources := &metav1.APIResourceList{
		APIResources: []metav1.APIResource{
			{Name: "widgets", Kind: "Widget"},
		},
	}
	wantResources := []*metav1.APIResourceList{fakeServerResources}

	fakeServerGroupsCache := []byte(`{"groups":[{"name":"foo","versions":[{"groupVersion":"foo/v1","version":"v1"}],"preferredVersion":{"groupVersion":"foo/v1","version":"v1"}}]}`)
	fakeServerResourcesCache := []byte(`{"resources":[{"name":"widgets","singularName":"","namespaced":false,"kind":"Widget","verbs":null}]}`)

	for _, tt := range []struct {
		name           string
		delegateClient discovery.DiscoveryInterface
		assets         map[string][]byte
		wantGroups     []*metav1.APIGroup
		wantResources  []*metav1.APIResourceList
		wantErr        string
	}{
		{
			name: "no error from delegate client",
			delegateClient: &mock_discovery.FakeDiscoveryClient{
				FakeServerGroups:    fakeServerGroups,
				FakeServerResources: fakeServerResources,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerGroups in delegate client, but ServerGroups cache exists",
			delegateClient: &mock_discovery.FakeDiscoveryClient{
				FakeServerResources: fakeServerResources,
			},
			assets: map[string][]byte{
				"servergroups.json": fakeServerGroupsCache,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, but ServerResourcesForGroupVersion cache exists",
			delegateClient: &mock_discovery.FakeDiscoveryClient{
				FakeServerGroups: fakeServerGroups,
			},
			assets: map[string][]byte{
				"foo/v1/serverresources.json": fakeServerResourcesCache,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name:           "error from ServerGroups in delegate client, cache doesn't exists",
			delegateClient: &mock_discovery.FakeDiscoveryClient{},
			wantErr:        "error from ServerGroups",
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, cache doesn't exists",
			delegateClient: &mock_discovery.FakeDiscoveryClient{
				FakeServerGroups: fakeServerGroups,
			},
			wantGroups:    wantGroups,
			wantResources: []*metav1.APIResourceList{},
			wantErr:       "unable to retrieve the complete list of server APIs: foo/v1: error from ServerResourcesForGroupVersion",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cli := &cacheFallbackDiscoveryClient{
				DiscoveryInterface: tt.delegateClient,
				log:                log,
				asset: func(name string) ([]byte, error) {
					if cache, ok := tt.assets[name]; ok {
						return cache, nil
					}

					return nil, fmt.Errorf("%s not found", name)
				},
			}

			groups, resources, err := cli.ServerGroupsAndResources()
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if !reflect.DeepEqual(tt.wantGroups, groups) {
				t.Error(cmp.Diff(groups, tt.wantGroups))
			}

			if !reflect.DeepEqual(tt.wantResources, resources) {
				t.Error(cmp.Diff(resources, tt.wantResources))
			}
		})
	}
}
