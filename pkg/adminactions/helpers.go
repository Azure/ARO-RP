package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (ka *adminactions) findGVR(apiresourcelist []*metav1.APIResourceList, groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	var matches []*schema.GroupVersionResource
	for _, apiresources := range apiresourcelist {
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
		return nil, api.NewCloudError(
			http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
			"", "The groupKind '%s' matched multiple groupKinds.", groupKind)
	}

	return matches[0], nil
}

func (a *adminactions) waitForPodRunning(ctx context.Context, pod *corev1.Pod) error {
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		pod, err := a.cli.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return pod.Status.Phase == corev1.PodRunning, nil
	}, ctx.Done())
}

func (a *adminactions) ensureNamespace(ns string) error {
	_, err := a.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (a *adminactions) applyConfigMap(cm *v1.ConfigMap) error {
	_, err := a.cli.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_cm, err := a.cli.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		cm.ResourceVersion = _cm.ResourceVersion
		_, err = a.cli.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
		return err
	})
}

func (a *adminactions) ApplySecret(s *v1.Secret) error {
	_, err := a.cli.CoreV1().Secrets(s.Namespace).Create(s)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_s, err := a.cli.CoreV1().Secrets(s.Namespace).Get(s.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		s.ResourceVersion = _s.ResourceVersion
		_, err = a.cli.CoreV1().Secrets(s.Namespace).Update(s)
		return err
	})
}

func (a *adminactions) ApplyAPIServerNamedServingCert(cert *configv1.APIServerNamedServingCert) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiserver, err := a.configcli.ConfigV1().APIServers().Get("cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		apiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{*cert}

		_, err = a.configcli.ConfigV1().APIServers().Update(apiserver)
		return err
	})
}

func (a *adminactions) ApplyIngressControllerCertificate(cert *v1.LocalObjectReference) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ic, err := a.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get("default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		ic.Spec.DefaultCertificate = cert

		_, err = a.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Update(ic)
		return err
	})
}

func (a *adminactions) applyServiceAccount(sa *v1.ServiceAccount) error {
	_, err := a.cli.CoreV1().ServiceAccounts(sa.Namespace).Create(sa)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_sa, err := a.cli.CoreV1().ServiceAccounts(sa.Namespace).Get(sa.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		sa.ResourceVersion = _sa.ResourceVersion
		_, err = a.cli.CoreV1().ServiceAccounts(sa.Namespace).Update(sa)
		return err
	})
}

func (a *adminactions) applyDaemonSet(ds *appsv1.DaemonSet) error {
	_, err := a.cli.AppsV1().DaemonSets(ds.Namespace).Create(ds)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := a.cli.AppsV1().DaemonSets(ds.Namespace).Get(ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = a.cli.AppsV1().DaemonSets(ds.Namespace).Update(ds)
		return err
	})
}
