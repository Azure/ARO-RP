package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	utildiscovery "github.com/Azure/ARO-RP/pkg/util/dynamichelper/discovery"
)

type GVRResolver interface {
	Refresh() error
	Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error)
}

type gvrResolver struct {
	log *logrus.Entry

	discovery    discovery.DiscoveryInterface
	apiresources []*metav1.APIResourceList
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
	_, r.apiresources, err = r.discovery.ServerGroupsAndResources()
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

	var matches []*schema.GroupVersionResource
	for _, apiresources := range r.apiresources {
		gv, err := schema.ParseGroupVersion(apiresources.GroupVersion)
		if err != nil {
			// this returns a fmt.Errorf which will result in a 500
			// in this case, this seems correct as the GV in kubernetes is wrong
			return nil, err
		}
		if optionalVersion != "" && gv.Version != optionalVersion {
			continue
		}
		for _, apiresource := range apiresources.APIResources {
			if strings.ContainsRune(apiresource.Name, '/') { // no subresources
				continue
			}

			gk := schema.GroupKind{
				Group: gv.Group,
				Kind:  apiresource.Kind,
			}

			if strings.EqualFold(gk.String(), groupKind) {
				return &schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: apiresource.Name,
				}, nil
			}

			if strings.EqualFold(apiresource.Kind, groupKind) {
				matches = append(matches, &schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: apiresource.Name,
				})
			}
		}
	}

	if len(matches) == 0 {
		return nil, api.NewCloudError(
			http.StatusBadRequest, api.CloudErrorCodeNotFound,
			"", "The groupKind '%s' was not found.", groupKind)
	}

	if len(matches) > 1 {
		var matchesGK []string
		for _, match := range matches {
			matchesGK = append(matchesGK, groupKind+"."+match.Group)
		}
		return nil, api.NewCloudError(
			http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
			"", "The groupKind '%s' matched multiple groupKinds (%s).", groupKind, strings.Join(matchesGK, ", "))
	}

	return matches[0], nil
}
