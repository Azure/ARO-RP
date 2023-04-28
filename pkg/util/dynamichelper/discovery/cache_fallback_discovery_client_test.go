package discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var _ discovery.DiscoveryInterface = &FakeDiscoveryClient{}

// TestVersion makes sure that bindata contains cache generated with the
// supported OpenShift version.
func TestVersion(t *testing.T) {
	file, err := embeddedFiles.Open("cache/assets_version")
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	assetsVersion := strings.TrimSuffix(string(b), "\n")
	if assetsVersion != version.DefaultInstallStream.Version.String() {
		t.Error("discovery cache is out of date: run make discoverycache")
	}
}

//go:embed test_cache
var testCache embed.FS

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

	for _, tt := range []struct {
		name           string
		delegateClient discovery.DiscoveryInterface
		wantGroups     []*metav1.APIGroup
		wantResources  []*metav1.APIResourceList
		wantErr        string
		noCache        bool
	}{
		{
			name: "no error from delegate client",
			delegateClient: &FakeDiscoveryClient{
				fakeServerGroups:    fakeServerGroups,
				fakeServerResources: fakeServerResources,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerGroups in delegate client, but ServerGroups cache exists",
			delegateClient: &FakeDiscoveryClient{
				fakeServerResources: fakeServerResources,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, but ServerResourcesForGroupVersion cache exists",
			delegateClient: &FakeDiscoveryClient{
				fakeServerGroups: fakeServerGroups,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name:           "error from ServerGroups in delegate client, cache doesn't exists",
			delegateClient: &FakeDiscoveryClient{},
			wantErr:        "unable to retrieve the complete list of server APIs: ",
			noCache:        true,
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, cache doesn't exists",
			delegateClient: &FakeDiscoveryClient{
				fakeServerGroups: fakeServerGroups,
			},
			wantGroups:    wantGroups,
			wantResources: []*metav1.APIResourceList{},
			wantErr:       "unable to retrieve the complete list of server APIs: foo/v1: error from ServerResourcesForGroupVersion",
			noCache:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cli := &cacheFallbackDiscoveryClient{
				DiscoveryInterface: tt.delegateClient,
				log:                log,
				cache:              testCache,
				cacheDir:           "test_cache",
			}
			if tt.noCache {
				cli.cacheDir = "not_there"
			}

			groups, resources, err := cli.ServerGroupsAndResources()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(tt.wantGroups, groups) {
				t.Error(cmp.Diff(groups, tt.wantGroups))
			}

			if !reflect.DeepEqual(tt.wantResources, resources) {
				t.Error(cmp.Diff(resources, tt.wantResources))
			}
		})
	}
}
