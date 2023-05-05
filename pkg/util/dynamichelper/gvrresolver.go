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

// GVRInterface defines the interface for refreshing and resolving Groups and
// Versions and Resources from the kubenetes api
type GVRInterface interface {
	Refresh() error
	Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error)
}

type gvrResolver struct {
	log *logrus.Entry

	discovery    discovery.DiscoveryInterface
	apiresources []*restmapper.APIGroupResources
}

// NewGVRResolver returns a new GVRResolver connected to the kubenernetes API using
// either the provided restconfig or using a discovery client with hardcoded
// configuration settings.
func NewGVRResolver(log *logrus.Entry, restconfig *rest.Config) (GVRInterface, error) {
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

// Refresh will refresh the list of known Group Resources from the kubernetes api
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

// Resolve returns, if possible, the GroupVersionResource as specified by the
// groupKind and optionalVersion provided
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
