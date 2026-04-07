package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"time"

	_ "embed"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/holmes"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

//go:embed staticresources/holmes-config.yaml
var holmesConfigYAML string

// InvestigateCluster creates an investigation pod on the Hive cluster, streams its logs, and cleans up.
// It accepts kubeconfig bytes, creates a temporary secret to hold them, and removes
// the secret (along with the pod and configmap) when the investigation completes.
func (hr *clusterManager) InvestigateCluster(ctx context.Context, hiveNamespace string, kubeconfig []byte, holmesConfig *holmes.HolmesConfig, question string, w io.Writer) error {
	id := uuid.New().String()[:8]
	configMapName := "holmes-config-" + id
	podName := "holmes-investigate-" + id
	kubeconfigSecretName := "holmes-kubeconfig-" + id

	hr.log.Infof("starting Holmes investigation %s in namespace %s", id, hiveNamespace)

	// Ensure cleanup of the secret, ConfigMap, and pod on exit.
	defer func() {
		cleanupCtx := context.Background()

		hr.log.Infof("cleaning up investigation pod %s", podName)
		err := hr.kubernetescli.CoreV1().Pods(hiveNamespace).Delete(cleanupCtx, podName, metav1.DeleteOptions{})
		if err != nil {
			hr.log.Warningf("failed to delete investigation pod %s: %v", podName, err)
		}

		hr.log.Infof("cleaning up investigation configmap %s", configMapName)
		err = hr.kubernetescli.CoreV1().ConfigMaps(hiveNamespace).Delete(cleanupCtx, configMapName, metav1.DeleteOptions{})
		if err != nil {
			hr.log.Warningf("failed to delete investigation configmap %s: %v", configMapName, err)
		}

		hr.log.Infof("cleaning up investigation secret %s", kubeconfigSecretName)
		err = hr.kubernetescli.CoreV1().Secrets(hiveNamespace).Delete(cleanupCtx, kubeconfigSecretName, metav1.DeleteOptions{})
		if err != nil {
			hr.log.Warningf("failed to delete investigation secret %s: %v", kubeconfigSecretName, err)
		}
	}()

	// 0. Create the temporary secret holding the kubeconfig.
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeconfigSecretName,
			Namespace: hiveNamespace,
		},
		Data: map[string][]byte{
			"config":            kubeconfig,
			"azure-api-key":     []byte(holmesConfig.AzureAPIKey),
			"azure-api-base":    []byte(holmesConfig.AzureAPIBase),
			"azure-api-version": []byte(holmesConfig.AzureAPIVersion),
		},
	}

	_, err := hr.kubernetescli.CoreV1().Secrets(hiveNamespace).Create(ctx, kubeconfigSecret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create investigation kubeconfig secret: %w", err)
	}

	// 1. Create the ConfigMap with Holmes toolsets config.
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: hiveNamespace,
		},
		Data: map[string]string{
			"config.yaml": holmesConfigYAML,
		},
	}

	_, err = hr.kubernetescli.CoreV1().ConfigMaps(hiveNamespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create investigation configmap: %w", err)
	}

	// 2. Create the investigation pod.
	activeDeadlineSeconds := int64(holmesConfig.DefaultTimeout)
	runAsUser := int64(1000)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: hiveNamespace,
		},
		Spec: corev1.PodSpec{
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			RestartPolicy:         corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "holmes",
					Image:   holmesConfig.Image,
					Command: []string{"python", "holmes_cli.py"},
					Args:    []string{"ask", question, "-n", "--model=" + holmesConfig.Model, "--config=/etc/holmes/config.yaml"},
					Env: []corev1.EnvVar{
						{
							Name: "AZURE_API_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: kubeconfigSecretName},
									Key:                  "azure-api-key",
								},
							},
						},
						{
							Name: "AZURE_API_BASE",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: kubeconfigSecretName},
									Key:                  "azure-api-base",
								},
							},
						},
						{
							Name: "AZURE_API_VERSION",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: kubeconfigSecretName},
									Key:                  "azure-api-version",
								},
							},
						},
						{
							Name:  "KUBECONFIG",
							Value: "/etc/kubeconfig/config",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kubeconfig",
							MountPath: "/etc/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      "holmes-config",
							MountPath: "/etc/holmes/config.yaml",
							SubPath:   "config.yaml",
							ReadOnly:  true,
						},
						{
							Name:      "tmp",
							MountPath: "/tmp",
						},
						{
							Name:      "holmes-cache",
							MountPath: "/.holmes",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:                &runAsUser,
						RunAsNonRoot:             pointerutils.ToPtr(true),
						AllowPrivilegeEscalation: pointerutils.ToPtr(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: kubeconfigSecretName,
						},
					},
				},
				{
					Name: "holmes-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapName,
							},
						},
					},
				},
				{
					Name: "tmp",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "holmes-cache",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	_, err = hr.kubernetescli.CoreV1().Pods(hiveNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create investigation pod: %w", err)
	}

	// 3. Wait for the pod to be running.
	err = hr.waitForPodRunning(ctx, hiveNamespace, podName, 60*time.Second)
	if err != nil {
		return fmt.Errorf("failed waiting for investigation pod to start: %w", err)
	}

	// 4. Stream pod logs.
	err = hr.streamPodLogs(ctx, hiveNamespace, podName, w)
	if err != nil {
		return fmt.Errorf("failed to stream investigation pod logs: %w", err)
	}

	return nil
}

func (hr *clusterManager) waitForPodRunning(ctx context.Context, namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pod, err := hr.kubernetescli.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod %s: %w", name, err)
		}

		switch pod.Status.Phase {
		case corev1.PodRunning, corev1.PodSucceeded:
			return nil
		case corev1.PodFailed:
			reason := pod.Status.Reason
			message := pod.Status.Message
			if len(pod.Status.ContainerStatuses) > 0 {
				cs := pod.Status.ContainerStatuses[0]
				if cs.State.Terminated != nil {
					reason = cs.State.Terminated.Reason
					message = cs.State.Terminated.Message
				} else if cs.State.Waiting != nil {
					reason = cs.State.Waiting.Reason
					message = cs.State.Waiting.Message
				}
			}
			return fmt.Errorf("pod %s failed: reason=%s message=%s", name, reason, message)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for pod %s to be running", name)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (hr *clusterManager) streamPodLogs(ctx context.Context, namespace, name string, w io.Writer) error {
	req := hr.kubernetescli.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{
		Follow: true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open log stream for pod %s: %w", name, err)
	}
	defer stream.Close()

	// Use a buffered reader to read line-by-line and flush after each write
	// so the client sees output in real-time instead of only when the pod exits.
	flusher, canFlush := w.(interface{ Flush() })
	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write log stream for pod %s: %w", name, writeErr)
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("failed to read log stream for pod %s: %w", name, readErr)
		}
	}

	return nil
}
