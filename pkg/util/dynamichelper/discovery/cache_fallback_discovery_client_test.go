package discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

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
			delegateClient: &fakeDiscoveryClient{
				fakeServerGroups:    fakeServerGroups,
				fakeServerResources: fakeServerResources,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerGroups in delegate client, but ServerGroups cache exists",
			delegateClient: &fakeDiscoveryClient{
				fakeServerResources: fakeServerResources,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, but ServerResourcesForGroupVersion cache exists",
			delegateClient: &fakeDiscoveryClient{
				fakeServerGroups: fakeServerGroups,
			},
			wantGroups:    wantGroups,
			wantResources: wantResources,
		},
		{
			name:           "error from ServerGroups in delegate client, cache doesn't exists",
			delegateClient: &fakeDiscoveryClient{},
			wantErr:        "error from ServerGroups",
			noCache:        true,
		},
		{
			name: "error from ServerResourcesForGroupVersion in delegate client, cache doesn't exists",
			delegateClient: &fakeDiscoveryClient{
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
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err, tt.wantErr)
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

type fakeDiscoveryClient struct {
	fakeServerGroups    *metav1.APIGroupList
	fakeServerResources *metav1.APIResourceList
}

var _ discovery.DiscoveryInterface = &fakeDiscoveryClient{}

func (c *fakeDiscoveryClient) RESTClient() rest.Interface {
	return nil
}

func (c *fakeDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	if c.fakeServerGroups != nil {
		return c.fakeServerGroups, nil
	}
	return nil, errors.New("error from ServerGroups")
}

func (c *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	if c.fakeServerResources != nil {
		return c.fakeServerResources, nil
	}

	return nil, errors.New("error from ServerResourcesForGroupVersion")
}

// Deprecated: use ServerGroupsAndResources instead.
func (c *fakeDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	_, rs, err := c.ServerGroupsAndResources()
	return rs, err
}

func (c *fakeDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	gs, _ := c.ServerGroups()
	resultGroups := []*metav1.APIGroup{}
	for i := range gs.Groups {
		resultGroups = append(resultGroups, &gs.Groups[i])
	}

	return resultGroups, []*metav1.APIResourceList{}, nil
}

func (c *fakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *fakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *fakeDiscoveryClient) ServerVersion() (*kversion.Info, error) {
	return &kversion.Info{}, nil
}

func (c *fakeDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return &openapi_v2.Document{}, nil
}
