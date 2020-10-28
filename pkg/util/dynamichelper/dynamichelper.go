package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	kadiscovery "github.com/Azure/ARO-RP/pkg/util/dynamichelper/discovery"
)

type DynamicHelper interface {
	RefreshAPIResources() error
	CreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	Delete(ctx context.Context, groupKind, namespace, name string) error
	Ensure(ctx context.Context, o *unstructured.Unstructured) error
	Get(ctx context.Context, groupKind, namespace, name string) (*unstructured.Unstructured, error)
	List(ctx context.Context, groupKind, namespace string) (*unstructured.UnstructuredList, error)
}

type dynamicHelper struct {
	log *logrus.Entry

	restconfig   *rest.Config
	dyn          dynamic.Interface
	apiresources []*metav1.APIResourceList
}

func New(log *logrus.Entry, restconfig *rest.Config) (DynamicHelper, error) {
	dh := &dynamicHelper{
		log:        log,
		restconfig: restconfig,
	}

	var err error
	dh.dyn, err = dynamic.NewForConfig(dh.restconfig)
	if err != nil {
		return nil, err
	}

	return dh, nil
}

func (dh *dynamicHelper) RefreshAPIResources() error {
	var cli discovery.DiscoveryInterface
	cli, err := discovery.NewDiscoveryClientForConfig(dh.restconfig)
	if err != nil {
		return err
	}
	cli = kadiscovery.NewCacheFallbackDiscoveryClient(dh.log, cli)

	_, dh.apiresources, err = cli.ServerGroupsAndResources()
	return err
}

func (dh *dynamicHelper) findGVR(groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	if dh.apiresources == nil {
		err := dh.RefreshAPIResources()
		if err != nil {
			return nil, err
		}
	}

	var matches []*schema.GroupVersionResource
	for _, apiresources := range dh.apiresources {
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

// CreateOrUpdate does nothing more than an Update call (and a Create if that
// call returned 404).  We don't add any fancy behaviour because this is called
// from the Geneva Admin context and we don't want to get in the SRE's way.
func (dh *dynamicHelper) CreateOrUpdate(ctx context.Context, o *unstructured.Unstructured) error {
	gvr, err := dh.findGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Update(ctx, o, metav1.UpdateOptions{})
	if !errors.IsNotFound(err) {
		return err
	}

	_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Create(ctx, o, metav1.CreateOptions{})
	return err
}

func (dh *dynamicHelper) Delete(ctx context.Context, groupKind, namespace, name string) error {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Ensure is called by the operator deploy tool and individual controllers.  It
// is intended to ensure that an object matches a desired state.  It is tolerant
// of unspecified fields in the desired state (e.g. it will leave typically
// leave .status untouched).
func (dh *dynamicHelper) Ensure(ctx context.Context, o *unstructured.Unstructured) error {
	gvr, err := dh.findGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Get(ctx, o.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			dh.log.Printf("Create %s", keyFuncO(o))
			_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Create(ctx, o, metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		o, changed, diff, err := merge(existing, o)
		if err != nil || !changed {
			return err
		}

		dh.log.Printf("Update %s: %s", keyFuncO(o), diff)

		_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Update(ctx, o, metav1.UpdateOptions{})
		return err
	})
}

func (dh *dynamicHelper) Get(ctx context.Context, groupKind, namespace, name string) (*unstructured.Unstructured, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (dh *dynamicHelper) List(ctx context.Context, groupKind, namespace string) (*unstructured.UnstructuredList, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

func diff(existing, o *unstructured.Unstructured) string {
	if o.GroupVersionKind().GroupKind().String() == "Secret" { // Don't show a diff if kind is Secret
		return ""
	}

	return cmp.Diff(existing.Object, o.Object)
}

// merge merges delta onto base using ugorji/go/codec semantics.  It returns the
// newly merged object (the inputs are untouched) plus a flag indicating if a
// change took place and a printable diff as appropriate
func merge(base, delta *unstructured.Unstructured) (*unstructured.Unstructured, bool, string, error) {
	copy := base.DeepCopy()

	h := &codec.JsonHandle{
		MapKeyAsString: true,
	}

	var b []byte
	err := codec.NewEncoderBytes(&b, h).Encode(delta.Object)
	if err != nil {
		return nil, false, "", err
	}

	err = codec.NewDecoderBytes(b, h).Decode(&copy.Object)
	if err != nil {
		return nil, false, "", err
	}

	// all new objects have a null creationTimestamp that causes every object to
	// be updated.
	copy.SetCreationTimestamp(base.GetCreationTimestamp())

	status, found, err := unstructured.NestedMap(base.Object, "status")
	if err == nil && found {
		unstructured.SetNestedMap(copy.Object, status, "status")
	} else {
		// prevent empty status objects from causing problems
		unstructured.RemoveNestedField(copy.Object, "status")
	}

	return copy, !reflect.DeepEqual(base, copy), diff(base, copy), nil
}

func keyFuncO(o *unstructured.Unstructured) string {
	return keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())
}

func keyFunc(gk schema.GroupKind, namespace, name string) string {
	s := gk.String()
	if namespace != "" {
		s += "/" + namespace
	}
	s += "/" + name

	return s
}
