package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

type DynamicHelper interface {
	Get(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	List(ctx context.Context, groupKind, namespace string) ([]byte, error)
	CreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	Delete(ctx context.Context, groupKind, namespace, name string) error
	ToUnstructured(ro runtime.Object) (*unstructured.Unstructured, error)
}

type UpdatePolicy struct {
	LogChanges                    bool
	RetryOnConflict               bool
	IgnoreDefaults                bool
	RefreshAPIResourcesOnNotFound bool
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
	err := dh.refreshAPIResources()
	if err != nil {
		return nil, err
	}
	return dh, nil
}

func (dh *dynamicHelper) refreshAPIResources() error {
	cli, err := discovery.NewDiscoveryClientForConfig(dh.restconfig)
	if err != nil {
		return err
	}
	_, dh.apiresources, err = cli.ServerGroupsAndResources()
	if err != nil {
		return err
	}
	dh.dyn, err = dynamic.NewForConfig(dh.restconfig)
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

// ToUnstructured converts a runtime.Object into an Unstructured
func (dh *dynamicHelper) ToUnstructured(ro runtime.Object) (*unstructured.Unstructured, error) {
	obj, ok := ro.(*unstructured.Unstructured)
	if !ok {
		b, err := yaml.Marshal(ro)
		if err != nil {
			return nil, err
		}
		obj = &unstructured.Unstructured{}
		err = yaml.Unmarshal(b, obj)
		if err != nil {
			return nil, err
		}
	}

	cleanNewObject(*obj)
	return obj, nil
}

func (dh *dynamicHelper) retryableError(err error) bool {
	if err == nil {
		return false
	}
	switch typeOfErr := err.(type) {
	case (*api.CloudError):
		return (typeOfErr.Code == api.CloudErrorCodeNotFound)
	case (*discovery.ErrGroupDiscoveryFailed):
		return true
	default:
		return false
	}
}

func (dh *dynamicHelper) findGVRWithRefresh(groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	if !dh.updatePolicy.RefreshAPIResourcesOnNotFound {
		return dh.findGVR(groupKind, optionalVersion)
	}
	var gvr *schema.GroupVersionResource
	err := retry.OnError(wait.Backoff{Steps: 4, Duration: 30 * time.Second, Factor: 2.0}, dh.retryableError, func() error {
		// this is used at cluster start up when kinds are still getting
		// registered.
		var gvrErr error
		gvr, gvrErr = dh.findGVR(groupKind, optionalVersion)
		if dh.retryableError(gvrErr) {
			dh.log.Infof("refreshAPIResources retrying")
			if refErr := dh.refreshAPIResources(); refErr != nil {
				dh.log.Infof("refreshAPIResources error: %v", refErr)
				return refErr
			}
		}
		return gvrErr
	})
	return gvr, err
}

func (dh *dynamicHelper) CreateOrUpdate(ctx context.Context, o *unstructured.Unstructured) error {
	dh.log.Infof("CreateOrUpdate: %s", keyFuncO(o))
	gvr, err := dh.findGVRWithRefresh(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
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

		rv := existing.GetResourceVersion()

		if dh.updatePolicy.IgnoreDefaults {
			err := clean(*existing)
			if err != nil {
				return err
			}
			defaults(*existing)
		}

		if !dh.needsUpdate(existing, o) {
			return nil
		}

		dh.log.Info("Update " + keyFuncO(o))
		if dh.updatePolicy.LogChanges {
			dh.logDiff(existing, o)
		}

		o.SetResourceVersion(rv)
		_, err = dh.dyn.Resource(*gvr).Namespace(o.GetNamespace()).Update(o, metav1.UpdateOptions{})
		return err
	})

	return err
}

func (dh *dynamicHelper) needsUpdate(existing, o *unstructured.Unstructured) bool {
	if reflect.DeepEqual(*existing, *o) {
		return false
	}

	return true
}

func (dh *dynamicHelper) logDiff(existing, o *unstructured.Unstructured) bool {
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
