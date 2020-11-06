package routefix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	projectv1 "github.com/openshift/api/project/v1"
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
	kubeNamespace       = "openshift-azure-routefix"
	kubeServiceAccount  = "system:serviceaccount:" + kubeNamespace + ":default"
	routefixImageFormat = "%s.azurecr.io/routefix:c5c4a5db"
	shellScript         = `for ((;;))
do
  if ip route show cache | grep -q 'mtu 1450'; then
    ip route show cache
    ip route flush cache
  fi

  sleep 60
done`
)

type RouteFix interface {
	CreateOrUpdate(ctx context.Context) error
}

type routeFix struct {
	log *logrus.Entry
	env env.Interface

	cli    kubernetes.Interface
	seccli securityclient.Interface
}

func New(log *logrus.Entry, e env.Interface, cli kubernetes.Interface, seccli securityclient.Interface) RouteFix {
	return &routeFix{
		log: log,
		env: e,

		cli:    cli,
		seccli: seccli,
	}
}

func (i *routeFix) routefixImage() string {
	return fmt.Sprintf(routefixImageFormat, i.env.ACRName())
}

func (i *routeFix) ensureNamespace(ctx context.Context, ns string) error {
	_, err := i.cli.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ns,
			Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
		},
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ns, err := i.cli.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
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

		_, err = i.cli.CoreV1().Namespaces().Update(ctx, _ns, metav1.UpdateOptions{})
		return err
	})
}

func (i *routeFix) applyDaemonSet(ctx context.Context, ds *appsv1.DaemonSet) error {
	_, err := i.cli.AppsV1().DaemonSets(ds.Namespace).Create(ctx, ds, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := i.cli.AppsV1().DaemonSets(ds.Namespace).Get(ctx, ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = i.cli.AppsV1().DaemonSets(ds.Namespace).Update(ctx, ds, metav1.UpdateOptions{})
		return err
	})
}

func (i *routeFix) CreateOrUpdate(ctx context.Context) error {
	err := i.ensureNamespace(ctx, kubeNamespace)
	if err != nil {
		return err
	}

	i.log.Print("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var scc *securityv1.SecurityContextConstraints
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		scc, err = i.seccli.SecurityV1().SecurityContextConstraints().Get(ctx, "privileged", metav1.GetOptions{})
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-routefix",
	}
	scc.Groups = nil
	scc.Users = []string{kubeServiceAccount}

	_, err = i.seccli.SecurityV1().SecurityContextConstraints().Create(ctx, scc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return i.applyDaemonSet(ctx, &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "routefix",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "routefix"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "routefix"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "routefix",
							Image: i.routefixImage(),
							Args: []string{
								"sh",
								"-c",
								shellScript,
							},
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
