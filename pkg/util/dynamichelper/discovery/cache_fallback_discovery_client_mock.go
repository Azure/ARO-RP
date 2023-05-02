package discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

type FakeDiscoveryClient struct {
	fakeServerGroups    *metav1.APIGroupList
	fakeServerResources *metav1.APIResourceList
}

func (c *FakeDiscoveryClient) RESTClient() rest.Interface {
	return nil
}

func (c *FakeDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	if c.fakeServerGroups != nil {
		return c.fakeServerGroups, nil
	}
	return nil, &discovery.ErrGroupDiscoveryFailed{}
}

func (c *FakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	if c.fakeServerResources != nil {
		return c.fakeServerResources, nil
	}

	return nil, errors.New("error from ServerResourcesForGroupVersion")
}

// Deprecated: use ServerGroupsAndResources instead.
func (c *FakeDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	_, rs, err := c.ServerGroupsAndResources()
	return rs, err
}

func (c *FakeDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	gs, err := c.ServerGroups()
	if err != nil {
		return nil, nil, err
	}
	resultGroups := []*metav1.APIGroup{}
	for i := range gs.Groups {
		resultGroups = append(resultGroups, &gs.Groups[i])
	}

	return resultGroups, []*metav1.APIResourceList{}, nil
}

func (c *FakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *FakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *FakeDiscoveryClient) ServerVersion() (*kversion.Info, error) {
	return &kversion.Info{}, nil
}

func (c *FakeDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return &openapi_v2.Document{}, nil
}

func (c *FakeDiscoveryClient) IsGroupDiscoveryFailedError(err error) bool {
	_, ok := err.(*discovery.ErrGroupDiscoveryFailed)
	return err != nil && ok
}
