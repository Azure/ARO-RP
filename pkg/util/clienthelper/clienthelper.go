package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-test/deep"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type Interface interface {
	EnsureDeleted(ctx context.Context, gvk schema.GroupVersionKind, key types.NamespacedName) error
	Ensure(ctx context.Context, objs ...kruntime.Object) error
	GetOne(ctx context.Context, key types.NamespacedName, obj kruntime.Object) error
}

type clientHelper struct {
	log *logrus.Entry

	client client.Client
}

func New(log *logrus.Entry, restconfig *rest.Config) (Interface, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(restconfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, err
	}

	client, err := client.New(restconfig, client.Options{Mapper: mapper})
	if err != nil {
		return nil, err
	}
	return NewWithClient(log, client), nil
}

func NewWithClient(log *logrus.Entry, client client.Client) Interface {
	return &clientHelper{
		log:    log,
		client: client,
	}
}

func (ch *clientHelper) EnsureDeleted(ctx context.Context, gvk schema.GroupVersionKind, key types.NamespacedName) error {
	a := meta.AsPartialObjectMetadata(&metav1.ObjectMeta{
		Name:      key.Name,
		Namespace: key.Namespace,
	})
	a.SetGroupVersionKind(gvk)

	ch.log.Infof("Delete kind %s ns %s name %s", gvk.Kind, key.Namespace, key.Name)
	err := ch.client.Delete(ctx, a)
	if kerrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (ch *clientHelper) GetOne(ctx context.Context, key types.NamespacedName, obj kruntime.Object) error {
	newObj, ok := obj.(client.Object)
	if !ok {
		return errors.New("can't convert object")
	}

	return ch.client.Get(ctx, key, newObj)
}

// Ensure that one or more objects match their desired state.  Only update
// objects that need to be updated.
func (ch *clientHelper) Ensure(ctx context.Context, objs ...kruntime.Object) error {
	for _, o := range objs {
		err := ch.ensureOne(ctx, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ch *clientHelper) ensureOne(ctx context.Context, new kruntime.Object) error {
	gvk, err := apiutil.GVKForObject(new, scheme.Scheme)
	if err != nil {
		return err
	}

	newObj, ok := new.(client.Object)
	if !ok {
		return fmt.Errorf("object of kind %s can't be made a client.Object", gvk.String())
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		old, err := scheme.Scheme.New(gvk)
		if err != nil {
			return err
		}

		oldObj, ok := old.(client.Object)
		if !ok {
			return fmt.Errorf("object of kind %s can't be made a client.Object", gvk.String())
		}

		err = ch.client.Get(ctx, client.ObjectKey{Namespace: newObj.GetNamespace(), Name: newObj.GetName()}, oldObj)
		if kerrors.IsNotFound(err) {
			ch.log.Infof("Create %s", keyFunc(gvk.GroupKind(), newObj.GetNamespace(), newObj.GetName()))
			return ch.client.Create(ctx, newObj)
		}
		if err != nil {
			return err
		}
		candidate, changed, diff, err := merge(oldObj, newObj)
		if err != nil || !changed {
			return err
		}
		ch.log.Infof("Update %s: %s", keyFunc(gvk.GroupKind(), candidate.GetNamespace(), candidate.GetName()), diff)
		return ch.client.Update(ctx, candidate)
	})
}

// merge takes the existing (old) and desired (new) objects.  It compares them
// to see if an update is necessary, fixes up the new object if needed, and
// returns the difference for debugging purposes.
func merge(old, new client.Object) (client.Object, bool, string, error) {
	if reflect.TypeOf(old) != reflect.TypeOf(new) {
		return nil, false, "", fmt.Errorf("types differ: %T %T", old, new)
	}

	// 1. Set defaults on new.  This gets rid of many false positive diffs.
	scheme.Scheme.Default(new)

	// 2. Copy immutable fields from old to new to avoid false positives.
	oldtypemeta := old.GetObjectKind()
	newtypemeta := new.GetObjectKind()

	newtypemeta.SetGroupVersionKind(oldtypemeta.GroupVersionKind())

	new.SetSelfLink(old.GetSelfLink())
	new.SetUID(old.GetUID())
	new.SetResourceVersion(old.GetResourceVersion())
	new.SetGeneration(old.GetGeneration())
	new.SetCreationTimestamp(old.GetCreationTimestamp())
	new.SetManagedFields(old.GetManagedFields())

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

	case *machinev1beta1.MachineHealthCheck:
		old, new := old.(*machinev1beta1.MachineHealthCheck), new.(*machinev1beta1.MachineHealthCheck)
		new.Status = old.Status
	}

	var diff string
	if _, ok := old.(*corev1.Secret); !ok { // Don't show a diff if kind is Secret
		diff = strings.Join(deep.Equal(old, new), "\n")
	} else {
		diff = "<scrubbed>"
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
