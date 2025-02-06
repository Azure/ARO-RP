package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type Interface interface {
	Refresh() error
	EnsureDeleted(ctx context.Context, groupKind, namespace, name string) error
	EnsureDeletedGVR(ctx context.Context, groupKind, namespace, name, optionalVersion string) error
	Ensure(ctx context.Context, objs ...kruntime.Object) error
	IsConstraintTemplateReady(ctx context.Context, name string) (bool, error)
}

type dynamicHelper struct {
	GVRResolver

	log           *logrus.Entry
	restcli       rest.Interface
	dynamicClient dynamic.Interface
}

func New(log *logrus.Entry, restconfig *rest.Config) (Interface, error) {
	dh := &dynamicHelper{
		log: log,
	}

	var err error
	dh.GVRResolver, err = NewGVRResolver(log, restconfig)
	if err != nil {
		return nil, err
	}

	dh.dynamicClient, err = dynamic.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	restconfig = rest.CopyConfig(restconfig)
	restconfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	restconfig.GroupVersion = &schema.GroupVersion{}

	dh.restcli, err = rest.RESTClientFor(restconfig)
	if err != nil {
		return nil, err
	}

	return dh, nil
}

func (dh *dynamicHelper) EnsureDeleted(ctx context.Context, groupKind, namespace, name string) error {
	return dh.EnsureDeletedGVR(ctx, groupKind, namespace, name, "")
}
func (dh *dynamicHelper) EnsureDeletedGVR(ctx context.Context, groupKind, namespace, name, optionalVersion string) error {
	gvr, err := dh.Resolve(groupKind, optionalVersion)
	if err != nil {
		return err
	}

	// gatekeeper policies are unstructured and should be deleted differently
	if isKindUnstructured(groupKind) {
		dh.log.Infof("Delete unstructured obj kind %s ns %s name %s version %s", groupKind, namespace, name, optionalVersion)
		return dh.deleteUnstructuredObj(ctx, groupKind, namespace, name)
	}
	dh.log.Infof("Delete kind %s ns %s name %s", groupKind, namespace, name)
	err = dh.restcli.Delete().AbsPath(makeURLSegments(gvr, namespace, name)...).Do(ctx).Error()
	if kerrors.IsNotFound(err) {
		err = nil
	}
	return err
}

// Ensure that one or more objects match their desired state.  Only update
// objects that need to be updated.
func (dh *dynamicHelper) Ensure(ctx context.Context, objs ...kruntime.Object) error {
	for _, o := range objs {
		if un, ok := o.(*unstructured.Unstructured); ok {
			err := dh.ensureUnstructuredObj(ctx, un)
			if err != nil {
				return err
			}
			continue
		}
		err := dh.ensureOne(ctx, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dh *dynamicHelper) ensureOne(ctx context.Context, new kruntime.Object) error {
	gvks, _, err := scheme.Scheme.ObjectKinds(new)
	if err != nil {
		return err
	}

	gvk := gvks[0]

	gvr, err := dh.Resolve(gvk.GroupKind().String(), gvk.Version)
	if err != nil {
		return err
	}

	acc, err := meta.Accessor(new)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		old, err := dh.restcli.Get().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), acc.GetName())...).Do(ctx).Get()
		if kerrors.IsNotFound(err) {
			dh.log.Infof("Create %s", keyFunc(gvk.GroupKind(), acc.GetNamespace(), acc.GetName()))
			return dh.restcli.Post().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), "")...).Body(new).Do(ctx).Error()
		}
		if err != nil {
			return err
		}
		candidate, changed, diff, err := dh.mergeWithLogic(acc.GetName(), gvk.GroupKind().String(), old, new)
		if err != nil || !changed {
			return err
		}
		dh.log.Infof("Update %s: %s", keyFunc(gvk.GroupKind(), acc.GetNamespace(), acc.GetName()), diff)
		return dh.restcli.Put().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), acc.GetName())...).Body(candidate).Do(ctx).Error()
	})
}

func (dh *dynamicHelper) mergeWithLogic(name, groupKind string, old, new kruntime.Object) (kruntime.Object, bool, string, error) {
	if strings.HasPrefix(name, "gatekeeper") {
		dh.log.Debugf("Skip updating %s: %s", name, groupKind)
		return nil, false, "", nil
	}
	if strings.HasPrefix(groupKind, "ConstraintTemplate.templates.gatekeeper") {
		return mergeGK(old, new)
	}

	return clienthelper.Merge(old.(client.Object), new.(client.Object))
}

func makeURLSegments(gvr *schema.GroupVersionResource, namespace, name string) (url []string) {
	if gvr.Group == "" {
		url = append(url, "api")
	} else {
		url = append(url, "apis", gvr.Group)
	}

	url = append(url, gvr.Version)

	if namespace != "" {
		url = append(url, "namespaces", namespace)
	}

	url = append(url, gvr.Resource)

	if len(name) > 0 {
		url = append(url, name)
	}

	return url
}
