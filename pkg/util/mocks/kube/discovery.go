package mock_discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
)

type FakeDiscoveryClient struct {
	FakeServerGroups    *metav1.APIGroupList
	FakeServerResources *metav1.APIResourceList
	Client              restclient.Interface
}

var _ discovery.DiscoveryInterface = &FakeDiscoveryClient{}

func (c *FakeDiscoveryClient) RESTClient() restclient.Interface {
	return c.Client
}

func (c *FakeDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	if c.FakeServerGroups != nil {
		return c.FakeServerGroups, nil
	}
	return nil, errors.New("error from ServerGroups")
}

func (c *FakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	if c.FakeServerResources != nil {
		return c.FakeServerResources, nil
	}

	return nil, errors.New("error from ServerResourcesForGroupVersion")
}

// Deprecated: use ServerGroupsAndResources instead.
func (c *FakeDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	_, rs, err := c.ServerGroupsAndResources()
	return rs, err
}

func (c *FakeDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	gs, _ := c.ServerGroups()
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
