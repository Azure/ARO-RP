package ifreload

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/env"
)

const (
	kubeNamespace       = "openshift-azure-ifreload"
	kubeServiceAccount  = "system:serviceaccount:" + kubeNamespace + ":default"
	ifreloadImageFormat = "%s.azurecr.io/ifreload:6dd7a73c"
)

type IfReload interface {
	CreateOrUpdate(ctx context.Context) error
}

type ifReload struct {
	log *logrus.Entry
	env env.Interface

	cli    kubernetes.Interface
	seccli securityclient.Interface
}

func New(log *logrus.Entry, e env.Interface, cli kubernetes.Interface, seccli securityclient.Interface) IfReload {
	return &ifReload{
		log: log,
		env: e,

		cli:    cli,
		seccli: seccli,
	}
}

func (i *ifReload) ifreloadImage() string {
	return fmt.Sprintf(ifreloadImageFormat, i.env.ACRName())
}

func (i *ifReload) ensureNamespace(ns string) error {
	_, err := i.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (i *ifReload) applyDaemonSet(ds *appsv1.DaemonSet) error {
	_, err := i.cli.AppsV1().DaemonSets(ds.Namespace).Create(ds)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := i.cli.AppsV1().DaemonSets(ds.Namespace).Get(ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = i.cli.AppsV1().DaemonSets(ds.Namespace).Update(ds)
		return err
	})
}

func (i *ifReload) CreateOrUpdate(ctx context.Context) error {
	err := i.ensureNamespace(kubeNamespace)
	if err != nil {
		return err
	}

	i.log.Print("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var scc *securityv1.SecurityContextConstraints
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		scc, err = i.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-ifreload",
	}
	scc.Groups = nil
	scc.Users = []string{kubeServiceAccount}

	_, err = i.seccli.SecurityV1().SecurityContextConstraints().Create(scc)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return i.applyDaemonSet(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ifreload",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "ifreload"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "ifreload"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "ifreload",
							Image: i.ifreloadImage(),
							// TODO: specify requests/limits
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
							},
						},
					},
					HostNetwork: true,
					Tolerations: []v1.Toleration{
						{
							Effect:   v1.TaintEffectNoExecute,
							Operator: v1.TolerationOpExists,
						},
						{
							Effect:   v1.TaintEffectNoSchedule,
							Operator: v1.TolerationOpExists,
						},
					},
				},
			},
		},
	})
}
