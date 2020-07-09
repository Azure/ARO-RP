package ensure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	projectv1 "github.com/openshift/api/project/v1"
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type Interface interface {
	Namespace(ns string) error
	SccGet() (*securityv1.SecurityContextConstraints, error)
	DaemonSet(ds *appsv1.DaemonSet) error
	ConfigMap(cm *v1.ConfigMap) error
	Secret(s *v1.Secret) error
	ServiceAccount(sa *v1.ServiceAccount) error
	SccCreate(scc *securityv1.SecurityContextConstraints) error
}

type ensure struct {
	cli    kubernetes.Interface
	seccli securityclient.Interface
}

var _ Interface = (*ensure)(nil)

func New(cli kubernetes.Interface, seccli securityclient.Interface) Interface {
	return &ensure{
		cli:    cli,
		seccli: seccli,
	}
}

func (e *ensure) ConfigMap(cm *v1.ConfigMap) error {
	_, err := e.cli.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_cm, err := e.cli.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		cm.ResourceVersion = _cm.ResourceVersion
		_, err = e.cli.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
		return err
	})
}

func (e *ensure) DaemonSet(ds *appsv1.DaemonSet) error {
	_, err := e.cli.AppsV1().DaemonSets(ds.Namespace).Create(ds)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := e.cli.AppsV1().DaemonSets(ds.Namespace).Get(ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = e.cli.AppsV1().DaemonSets(ds.Namespace).Update(ds)
		return err
	})
}

func (e *ensure) Secret(s *v1.Secret) error {
	_, err := e.cli.CoreV1().Secrets(s.Namespace).Create(s)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_s, err := e.cli.CoreV1().Secrets(s.Namespace).Get(s.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		s.ResourceVersion = _s.ResourceVersion
		_, err = e.cli.CoreV1().Secrets(s.Namespace).Update(s)
		return err
	})
}

func (e *ensure) ServiceAccount(sa *v1.ServiceAccount) error {
	_, err := e.cli.CoreV1().ServiceAccounts(sa.Namespace).Create(sa)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_sa, err := e.cli.CoreV1().ServiceAccounts(sa.Namespace).Get(sa.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		sa.ResourceVersion = _sa.ResourceVersion
		_, err = e.cli.CoreV1().ServiceAccounts(sa.Namespace).Update(sa)
		return err
	})
}
func (e *ensure) Namespace(ns string) error {
	_, err := e.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ns,
			Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ns, err := e.cli.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if _ns.Annotations == nil {
			_ns.Annotations = map[string]string{}
		}

		if annotation, ok := _ns.Annotations[projectv1.ProjectNodeSelector]; annotation == "" && ok {
			return nil
		}
		_ns.Annotations[projectv1.ProjectNodeSelector] = ""
		_, err = e.cli.CoreV1().Namespaces().Update(_ns)
		return err
	})
}

func (e *ensure) SccGet() (*securityv1.SecurityContextConstraints, error) {
	scc, err := e.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return scc, nil
}

func (e *ensure) SccCreate(scc *securityv1.SecurityContextConstraints) error {
	_, err := e.seccli.SecurityV1().SecurityContextConstraints().Create(scc)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_scc, err := e.seccli.SecurityV1().SecurityContextConstraints().Get(scc.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		scc.ResourceVersion = _scc.ResourceVersion
		_, err = e.seccli.SecurityV1().SecurityContextConstraints().Update(scc)
		return err
	})
}
