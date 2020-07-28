package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	kadiscovery "github.com/Azure/ARO-RP/pkg/dynamichelper/discovery"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

type DynamicHelper interface {
	RefreshAPIResources() error
	CreateOrUpdate(obj *unstructured.Unstructured) error
	Delete(groupKind, namespace, name string) error
	Get(groupKind, namespace, name string) (*unstructured.Unstructured, error)
	List(groupKind, namespace string) (*unstructured.UnstructuredList, error)
}

type UpdatePolicy struct {
	LogChanges              bool
	RetryOnConflict         bool
	AvoidUnnecessaryUpdates bool
}

type dynamicHelper struct {
	log *logrus.Entry

	updatePolicy UpdatePolicy

	restconfig   *rest.Config
	dyn          dynamic.Interface
	apiresources []*metav1.APIResourceList
}

func New(log *logrus.Entry, restconfig *rest.Config, updatePolicy UpdatePolicy) (DynamicHelper, error) {
	dh := &dynamicHelper{
		log:          log,
		updatePolicy: updatePolicy,
		restconfig:   restconfig,
	}

	var err error
	dh.dyn, err = dynamic.NewForConfig(dh.restconfig)
	if err != nil {
		return nil, err
	}

	err = dh.RefreshAPIResources()
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

func (dh *dynamicHelper) CreateOrUpdate(o *unstructured.Unstructured) error {
	gvr, err := dh.findGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return dh.updatePolicy.RetryOnConflict && apierrors.ReasonForError(err) == metav1.StatusReasonConflict
	}, func() error {
		existing, err := dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			dh.log.Info("Create " + keyFuncO(o))
			_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Create(o, metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		if dh.updatePolicy.AvoidUnnecessaryUpdates {
			copyImmutableFields(o, existing)

			if !dh.needsUpdate(reflect.ValueOf(existing.Object), reflect.ValueOf(o.Object)) {
				return nil
			}
		} else {
			o.SetResourceVersion(existing.GetResourceVersion())
		}

		if !dh.logDiff(existing, o) {
			dh.log.Info("Update ", keyFuncO(o))
		}

		_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Update(o, metav1.UpdateOptions{})
		return err
	})

	return err
}

func (dh *dynamicHelper) Delete(groupKind, namespace, name string) error {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (dh *dynamicHelper) Get(groupKind, namespace, name string) (*unstructured.Unstructured, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
}

func (dh *dynamicHelper) List(groupKind, namespace string) (*unstructured.UnstructuredList, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).List(metav1.ListOptions{})
}

// needsUpdate: the idea is that we recursively compare existing and o; when we
// get to a map, we only check if the keys that are set in o are identical in
// existing.  We don't pay any attention to keys in existing that o has no
// opinion about.
func (dh *dynamicHelper) needsUpdate(existing, o reflect.Value) bool {
	if existing.Type() != o.Type() {
		return true
	}

	switch o.Kind() {
	case reflect.Map:
		i := o.MapRange()
		for i.Next() {
			if dh.needsUpdate(existing.MapIndex(i.Key()), i.Value()) {
				return true
			}
		}
		return false

	case reflect.Interface, reflect.Ptr:
		if existing.IsNil() || o.IsNil() {
			return existing.IsNil() != o.IsNil()
		}

		return dh.needsUpdate(existing.Elem(), o.Elem())

	case reflect.Slice, reflect.Array:
		if existing.IsNil() || o.IsNil() {
			return existing.IsNil() != o.IsNil()
		}
		if o.Len() != existing.Len() {
			return true
		}
		for i := 0; i < o.Len(); i++ {
			if dh.needsUpdate(existing.Index(i), o.Index(i)) {
				return true
			}
		}
		return false

	default:
		return !reflect.DeepEqual(existing.Interface(), o.Interface())
	}
}

func (dh *dynamicHelper) logDiff(existing, o *unstructured.Unstructured) bool {
	gk := o.GroupVersionKind().GroupKind()
	diffShown := false
	if dh.updatePolicy.LogChanges && gk.String() != "Secret" { // Don't show a diff if kind is Secret
		if diff := cmp.Diff(*existing, *o); diff != "" {
			dh.log.Info("Update ", keyFuncO(o), diff)
			diffShown = true
		}
	}
	return diffShown
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
