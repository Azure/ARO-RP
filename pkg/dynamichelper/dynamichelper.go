package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/openshift/openshift-azure/pkg/util/cmp"
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
)

type DynamicHelper interface {
	Get(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	List(ctx context.Context, groupKind, namespace string) ([]byte, error)
	CreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	Delete(ctx context.Context, groupKind, namespace, name string) error
	UnmarshalYAML(b []byte) (unstructured.Unstructured, error)
}

type dynamicHelper struct {
	log *logrus.Entry

	logChanges      bool
	retryOnConflict bool

	restconfig *rest.Config

	dyn          dynamic.Interface
	apiresources []*metav1.APIResourceList
}

func New(log *logrus.Entry, restconfig *rest.Config, logChanges, retryOnConflict bool) (DynamicHelper, error) {
	dh := &dynamicHelper{
		log:             log,
		logChanges:      logChanges,
		retryOnConflict: retryOnConflict,
		restconfig:      restconfig,
	}
	cli, err := discovery.NewDiscoveryClientForConfig(dh.restconfig)
	if err != nil {
		return nil, err
	}

	_, dh.apiresources, err = cli.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	dh.dyn, err = dynamic.NewForConfig(dh.restconfig)
	if err != nil {
		return nil, err
	}
	return dh, nil
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
			http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
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

// unmarshal has to reimplement yaml.unmarshal because it universally mangles yaml
// integers into float64s, whereas the Kubernetes client library uses int64s
// wherever it can.  Such a difference can cause us to update objects when
// we don't actually need to.
func (dh *dynamicHelper) UnmarshalYAML(b []byte) (unstructured.Unstructured, error) {
	json, err := yaml.YAMLToJSON(b)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	var o unstructured.Unstructured
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(json, nil, &o)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return o, nil
}

func (dh *dynamicHelper) Get(ctx context.Context, groupKind, namespace, name string) ([]byte, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	un, err := dh.dyn.Resource(*gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

func (dh *dynamicHelper) List(ctx context.Context, groupKind, namespace string) ([]byte, error) {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	ul, err := dh.dyn.Resource(*gvr).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func (dh *dynamicHelper) CreateOrUpdate(ctx context.Context, o *unstructured.Unstructured) error {
	gvr, err := dh.findGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return dh.retryOnConflict && apierrors.ReasonForError(err) == metav1.StatusReasonConflict
	}, func() error {
		existing, err := dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			dh.log.Info("Create " + keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
			_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Create(o, metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		rv := existing.GetResourceVersion()

		if !dh.needsUpdate(existing, o) {
			return err
		}
		dh.logDiff(existing, o)

		o.SetResourceVersion(rv)
		_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Update(o, metav1.UpdateOptions{})
		return err
	})

	return err
}

func (dh *dynamicHelper) needsUpdate(existing, o *unstructured.Unstructured) bool {
	if o.GetKind() == "Namespace" {
		// don't need updating
		return false
	}

	if reflect.DeepEqual(*existing, *o) {
		return false
	}

	dh.log.Info("Update " + keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

	return true
}

func (dh *dynamicHelper) logDiff(existing, o *unstructured.Unstructured) bool {
	if !dh.logChanges {
		return false
	}
	// TODO: we should have tests that monitor these diffs:
	// 1) when a cluster is created
	// 2) when sync is run twice back-to-back on the same cluster

	// Don't show a diff if kind is Secret
	gk := o.GroupVersionKind().GroupKind()
	diffShown := false
	if gk.String() != "Secret" {
		if diff := cmp.Diff(*existing, *o); diff != "" {
			dh.log.Info(diff)
			diffShown = true
		}
	}
	return diffShown
}

func (dh *dynamicHelper) Delete(ctx context.Context, groupKind, namespace, name string) error {
	gvr, err := dh.findGVR(groupKind, "")
	if err != nil {
		return err
	}

	return dh.dyn.Resource(*gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
}

func keyFunc(gk schema.GroupKind, namespace, name string) string {
	s := gk.String()
	if namespace != "" {
		s += "/" + namespace
	}
	s += "/" + name

	return s
}
