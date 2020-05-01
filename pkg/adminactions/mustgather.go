package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (a *adminactions) MustGather(ctx context.Context, w io.Writer) error {
	ns, err := a.cli.CoreV1().Namespaces().Create(&corev1.Namespace{
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
		err = a.cli.CoreV1().Namespaces().Delete(ns.Name, nil)
		if err != nil {
			a.log.Error(err)
		}
	}()

	crb, err := a.cli.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
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
		err = a.cli.RbacV1().ClusterRoleBindings().Delete(crb.Name, nil)
		if err != nil {
			a.log.Error(err)
		}
	}()

	pod, err := a.cli.CoreV1().Pods(ns.Name).Create(&corev1.Pod{
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
					Image: version.OpenShiftMustGather,
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
					Image: version.OpenShiftMustGather,
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

	a.log.Info("waiting for must-gather pod")
	err = a.waitForPodRunning(ctx, pod)
	if err != nil {
		return err
	}

	a.log.Info("must-gather pod running")
	rc, err := portforward.ExecStdout(ctx, a.env, a.oc, pod.Namespace, pod.Name, "copy", []string{"tar", "cz", "/must-gather"})
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(w, rc)
	return err
}
