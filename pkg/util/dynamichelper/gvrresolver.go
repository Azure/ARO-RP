package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	utildiscovery "github.com/Azure/ARO-RP/pkg/util/dynamichelper/discovery"
)

type GVRResolver interface {
	Refresh() error
	Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error)
}

type gvrResolver struct {
	log *logrus.Entry

	discovery    discovery.DiscoveryInterface
	apiresources []*restmapper.APIGroupResources
}

func NewGVRResolver(log *logrus.Entry, restconfig *rest.Config) (GVRResolver, error) {
	r := &gvrResolver{
		log: log,
	}

	var err error
	r.discovery, err = discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	r.discovery = utildiscovery.NewCacheFallbackDiscoveryClient(r.log, r.discovery)

	return r, nil
}

func (r *gvrResolver) Refresh() (err error) {
	r.apiresources, err = restmapper.GetAPIGroupResources(r.discovery)
	if discovery.IsGroupDiscoveryFailedError(err) {
		// Some group discovery failed; dh.apiresources will have all the ones
		// that worked. This error can happen with a misconfigured apiservice,
		// for example. Log it and try to keep going.
		r.log.Warn(err)
		return nil
	}
	return err
}

func (r *gvrResolver) Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	if r.apiresources == nil {
		err := r.Refresh()
		if err != nil {
			return nil, err
		}
	}

	mapper := restmapper.NewDiscoveryRESTMapper(r.apiresources)
	result, err := mapper.ResourceFor(schema.ParseGroupResource(groupKind).WithVersion(optionalVersion))
	if err != nil {
		return nil, err
	}
	return &result, err
}
