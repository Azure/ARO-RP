// Package dynamichelper interfaces with the kubernetes API and providers helpers
// for managing GVK (group, version, kind) api resources using dynamic types
// derived from the api.
//
// This helper exists so that developers do not have to statically create
// types for each group/version/kind that they need to interact with.
package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

// Interface defines a set of methods which allow retrieving and updating
// GVKs from the kubernets api.
type Interface interface {
	Refresh() error                                                             // Refresh the in-memory storage of the GVK
	EnsureDeleted(ctx context.Context, groupKind, namespace, name string) error // Ensures the deletion of the GVK
	Ensure(ctx context.Context, objs ...kruntime.Object) error                  // Ensures the GVK exists and is correct
}

type dynamicHelper struct {
	GVRResolver

	log     *logrus.Entry
	restcli rest.Interface
}

// New creates a new dynamichelper with the desired Interface
//
// Using the provided restconfig we connect to the cluster
// and return the dynamichelper.Interface.
func New(log *logrus.Entry, restconfig *rest.Config) (Interface, error) {
	dh := &dynamicHelper{
		log: log,
	}

	var err error
	dh.GVRResolver, err = NewGVRResolver(log, restconfig)
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

// EnsureDeleted uses the rest cluster connection to delete the GVK from
// the requested namespace
func (dh *dynamicHelper) EnsureDeleted(ctx context.Context, groupKind, namespace, name string) error {
	gvr, err := dh.Resolve(groupKind, "")
	if err != nil {
		return err
	}

	err = dh.restcli.Delete().AbsPath(makeURLSegments(gvr, namespace, name)...).Do(ctx).Error()
	if kerrors.IsNotFound(err) {
		err = nil
	}
	return err
}

// Ensure ensures that one or more objects match their desired state. Only updates
// objects that need to be updated. Assumes that all provided objects are fully-
// formed and contain a group, version and kind.
func (dh *dynamicHelper) Ensure(ctx context.Context, objs ...kruntime.Object) error {
	for _, o := range objs {
		err := dh.ensureOne(ctx, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dh *dynamicHelper) ensureOne(ctx context.Context, new kruntime.Object) error {
	// Grab all possible GVKs for object
	gvks, _, err := scheme.Scheme.ObjectKinds(new)
	if err != nil {
		return err
	}

	gvk := gvks[0]
	// ASSUMPTION: The above assumes there will only ever be one gvk for the passed-in object
	// but there may be a case where the object is not fully formed and therefore may match
	// multiple gvks. This is a code-path that we don't test for so it may cause unintended
	// side-effects. Therefore we log a warning for our future selves.
	if len(gvks) > 1 {
		_gvk := new.GetObjectKind().GroupVersionKind()
		dh.log.Warnf("While retrieving kinds for an object \"%s\", more that one version returned which is a codepath that's not fully handled.", _gvk.String())
	}

	// Retrieve the schema for the GVK
	gvr, err := dh.Resolve(gvk.GroupKind().String(), gvk.Version)
	if err != nil {
		return err
	}

	acc, err := meta.Accessor(new)
	if err != nil {
		return err
	}

	// Perform the update retrying if other clients are accessing the object
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the old object
		old, err := dh.restcli.Get().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), acc.GetName())...).Do(ctx).Get()
		if kerrors.IsNotFound(err) {
			dh.log.Printf("Create %s", keyFunc(gvk.GroupKind(), acc.GetNamespace(), acc.GetName()))
			return dh.restcli.Post().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), "")...).Body(new).Do(ctx).Error()
		}
		if err != nil {
			return err
		}

		// Merge the two objects
		new, changed, diff, err := merge(old, new)
		if err != nil || !changed {
			return err
		}

		dh.log.Printf("Update %s: %s", keyFunc(gvk.GroupKind(), acc.GetNamespace(), acc.GetName()), diff)
		// Update the object in the api
		return dh.restcli.Put().AbsPath(makeURLSegments(gvr, acc.GetNamespace(), acc.GetName())...).Body(new).Do(ctx).Error()
	})
}

// merge takes the existing (old) and desired (new) objects.  It compares them
// to see if an update is necessary, fixes up the new object if needed, and
// returns the difference for debugging purposes.
func merge(old, new kruntime.Object) (kruntime.Object, bool, string, error) {
	if reflect.TypeOf(old) != reflect.TypeOf(new) {
		return nil, false, "", fmt.Errorf("types differ: %T %T", old, new)
	}

	// 1. Set defaults on new.  This gets rid of many false positive diffs.
	scheme.Scheme.Default(new)

	// 2. Copy immutable fields from old to new to avoid false positives.
	oldtypemeta := old.GetObjectKind()
	newtypemeta := new.GetObjectKind()

	oldobjectmeta, err := meta.Accessor(old)
	if err != nil {
		return nil, false, "", err
	}

	newobjectmeta, err := meta.Accessor(new)
	if err != nil {
		return nil, false, "", err
	}

	newtypemeta.SetGroupVersionKind(oldtypemeta.GroupVersionKind())

	newobjectmeta.SetSelfLink(oldobjectmeta.GetSelfLink())
	newobjectmeta.SetUID(oldobjectmeta.GetUID())
	newobjectmeta.SetResourceVersion(oldobjectmeta.GetResourceVersion())
	newobjectmeta.SetGeneration(oldobjectmeta.GetGeneration())
	newobjectmeta.SetCreationTimestamp(oldobjectmeta.GetCreationTimestamp())
	newobjectmeta.SetManagedFields(oldobjectmeta.GetManagedFields())

	// 3. Do fix-ups on a per-Kind basis.
	switch old.(type) {
	case *corev1.Namespace:
		old, new := old.(*corev1.Namespace), new.(*corev1.Namespace)
		for _, name := range []string{
			"openshift.io/sa.scc.mcs",
			"openshift.io/sa.scc.supplemental-groups",
			"openshift.io/sa.scc.uid-range",
		} {
			copyAnnotation(&new.ObjectMeta, &old.ObjectMeta, name)
		}
		// Copy OLM label
		for k := range old.Labels {
			if strings.HasPrefix(k, "olm.operatorgroup.uid/") {
				copyLabel(&new.ObjectMeta, &old.ObjectMeta, k)
			}
		}
		new.Spec.Finalizers = old.Spec.Finalizers
		new.Status = old.Status

	case *corev1.ServiceAccount:
		old, new := old.(*corev1.ServiceAccount), new.(*corev1.ServiceAccount)
		new.Secrets = old.Secrets
		new.ImagePullSecrets = old.ImagePullSecrets

	case *corev1.Service:
		old, new := old.(*corev1.Service), new.(*corev1.Service)
		new.Spec.ClusterIP = old.Spec.ClusterIP

	case *appsv1.DaemonSet:
		old, new := old.(*appsv1.DaemonSet), new.(*appsv1.DaemonSet)
		copyAnnotation(&new.ObjectMeta, &old.ObjectMeta, "deprecated.daemonset.template.generation")
		new.Status = old.Status

	case *appsv1.Deployment:
		old, new := old.(*appsv1.Deployment), new.(*appsv1.Deployment)
		copyAnnotation(&new.ObjectMeta, &old.ObjectMeta, "deployment.kubernetes.io/revision")

		// populated automatically by the Kubernetes API (observed on 4.9)
		if old.Spec.Template.Spec.DeprecatedServiceAccount != "" {
			new.Spec.Template.Spec.DeprecatedServiceAccount = old.Spec.Template.Spec.DeprecatedServiceAccount
		}

		new.Status = old.Status

	case *mcv1.KubeletConfig:
		old, new := old.(*mcv1.KubeletConfig), new.(*mcv1.KubeletConfig)
		new.Status = old.Status

	case *extensionsv1beta1.CustomResourceDefinition:
		old, new := old.(*extensionsv1beta1.CustomResourceDefinition), new.(*extensionsv1beta1.CustomResourceDefinition)
		new.Status = old.Status

	case *extensionsv1.CustomResourceDefinition:
		old, new := old.(*extensionsv1.CustomResourceDefinition), new.(*extensionsv1.CustomResourceDefinition)
		new.Status = old.Status

	case *arov1alpha1.Cluster:
		old, new := old.(*arov1alpha1.Cluster), new.(*arov1alpha1.Cluster)
		new.Status = old.Status

	case *hivev1.ClusterDeployment:
		old, new := old.(*hivev1.ClusterDeployment), new.(*hivev1.ClusterDeployment)
		new.ObjectMeta.Finalizers = old.ObjectMeta.Finalizers
		new.Status = old.Status

	case *corev1.ConfigMap:
		old, new := old.(*corev1.ConfigMap), new.(*corev1.ConfigMap)

		_, injectTrustBundle := new.ObjectMeta.Labels["config.openshift.io/inject-trusted-cabundle"]
		if injectTrustBundle {
			caBundle, ext := old.Data["ca-bundle.crt"]
			if ext {
				new.Data["ca-bundle.crt"] = caBundle
			}
		}
	}

	var diff string
	if _, ok := old.(*corev1.Secret); !ok { // Don't show a diff if kind is Secret
		diff = cmp.Diff(old, new)
	}

	return new, !reflect.DeepEqual(old, new), diff, nil
}

func copyAnnotation(dst, src *metav1.ObjectMeta, name string) {
	if _, found := src.Annotations[name]; found {
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations[name] = src.Annotations[name]
	}
}

func copyLabel(dst, src *metav1.ObjectMeta, name string) {
	if _, found := src.Labels[name]; found {
		if dst.Labels == nil {
			dst.Labels = map[string]string{}
		}
		dst.Labels[name] = src.Labels[name]
	}
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
