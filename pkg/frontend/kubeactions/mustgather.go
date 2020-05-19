package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (ka *kubeactions) MustGather(ctx context.Context, oc *api.OpenShiftCluster, w io.Writer) error {
	restconfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	ns, err := cli.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "openshift-must-gather-",
			Labels: map[string]string{
				"openshift.io/run-level": "0",
			},
		},
	})
	if err != nil {
		return err
	}

	defer func() {
		err = cli.CoreV1().Namespaces().Delete(ns.Name, nil)
		if err != nil {
			ka.log.Error(err)
		}
	}()

	crb, err := cli.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "must-gather-",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: ns.Name,
			},
		},
	})
	if err != nil {
		return err
	}

	defer func() {
		err = cli.RbacV1().ClusterRoleBindings().Delete(crb.Name, nil)
		if err != nil {
			ka.log.Error(err)
		}
	}()

	pod, err := cli.CoreV1().Pods(ns.Name).Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "must-gather",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "must-gather-output",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				{
					Name:  "gather",
					Image: version.InstallStream.MustGather, // TODO(mjudeikis): We should try get y-version image first and default to latest if not found.
					Command: []string{
						"/usr/bin/gather",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "must-gather-output",
							MountPath: "/must-gather",
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "copy",
					Image: version.InstallStream.MustGather,
					Command: []string{
						"sleep",
						"infinity",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "must-gather-output",
							MountPath: "/must-gather",
						},
					},
				},
			},
			TerminationGracePeriodSeconds: to.Int64Ptr(0),
			Tolerations: []corev1.Toleration{
				{
					Operator: "Exists",
				},
			},
		},
	})
	if err != nil {
		return err
	}

	ka.log.Info("waiting for must-gather pod")
	err = ka.waitForPodRunning(ctx, cli, pod)
	if err != nil {
		return err
	}

	ka.log.Info("must-gather pod running")
	rc, err := portforward.ExecStdout(ctx, ka.log, ka.env, oc, pod.Namespace, pod.Name, "copy", []string{"tar", "cz", "/must-gather"})
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(w, rc)
	return err
}

func (ka *kubeactions) waitForPodRunning(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) error {
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		pod, err := cli.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return pod.Status.Phase == corev1.PodRunning, nil
	}, ctx.Done())
}
