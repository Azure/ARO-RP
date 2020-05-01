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

	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (ka *kubeactions) MustGather(ctx context.Context, w io.Writer) error {
	ns, err := createMustGatherNamespace(ka)
	if err != nil {
		return err
	}

	defer func() {
		err = ka.kubernetescli.CoreV1().Namespaces().Delete(ns.Name, nil)
		if err != nil {
			ka.log.Error(err)
		}
	}()

	crb, err := createMustGatherClusterRoleBindings(ka, ns)
	if err != nil {
		return err
	}

	defer func() {
		err = ka.kubernetescli.RbacV1().ClusterRoleBindings().Delete(crb.Name, nil)
		if err != nil {
			ka.log.Error(err)
		}
	}()

	pod, err := createMustGatherPod(ka, ns)
	if err != nil {
		return err
	}

	err = waitForMustGatherPod(ctx, ka, pod)
	if err != nil {
		return err
	}

	return copyMustGatherLogs(ctx, ka, pod, w, portforward.NewExec(ka.env, ka.oc))
}

func createMustGatherNamespace(ka *kubeactions) (*corev1.Namespace, error) {
	return ka.kubernetescli.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "openshift-must-gather-",
			Labels: map[string]string{
				"openshift.io/run-level": "0",
			},
		},
	})
}

func createMustGatherClusterRoleBindings(ka *kubeactions, ns *corev1.Namespace) (*rbacv1.ClusterRoleBinding, error) {
	return ka.kubernetescli.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
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
}

// createMustGatherPod attempts to create a pod containing a volume for output, a container,
// for gathering logs and a container for copying the logs from the output volume
func createMustGatherPod(ka *kubeactions, ns *corev1.Namespace) (*corev1.Pod, error) {
	return ka.kubernetescli.CoreV1().Pods(ns.Name).Create(&corev1.Pod{
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
}

func waitForMustGatherPod(ctx context.Context, ka *kubeactions, pod *corev1.Pod) error {
	ka.log.Info("waiting for must-gather pod")
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		pod, err := ka.kubernetescli.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	}, ctx.Done())
	if err != nil {
		return err
	}
	ka.log.Info("must-gather pod running")
	return err
}

func copyMustGatherLogs(ctx context.Context, ka *kubeactions, pod *corev1.Pod, w io.Writer, e portforward.Exec) error {
	readCloser, err := e.Stdout(ctx, pod.Namespace, pod.Name, "copy", []string{"tar", "cz", "/must-gather"})
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(w, readCloser)
	return err
}
