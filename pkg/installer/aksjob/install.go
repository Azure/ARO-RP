package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/installer/inputs"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

// Install creates and runs an installer Job on AKS
func (m *manager) Install(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("starting AKS Job-based installation")

	s := []steps.Step{
		steps.Action(func(ctx context.Context) error {
			return m.ensureNamespace(ctx, installerInputs)
		}),
		steps.Action(func(ctx context.Context) error {
			return m.ensureServiceAccount(ctx, installerInputs)
		}),
		steps.Action(func(ctx context.Context) error {
			return m.ensureSecrets(ctx, installerInputs)
		}),
		steps.Action(func(ctx context.Context) error {
			return m.ensureJob(ctx, installerInputs)
		}),
		steps.Condition(func(ctx context.Context) (bool, error) {
			return m.jobCompleted(ctx, installerInputs)
		}, installerInputs.Timeout, false),
		steps.Action(func(ctx context.Context) error {
			return m.collectLogs(ctx, installerInputs)
		}),
		steps.Action(func(ctx context.Context) error {
			return m.Cleanup(ctx, installerInputs.Namespace)
		}),
	}

	_, err := steps.Run(ctx, m.log, 10*time.Second, s, nil, "")
	return err
}

// ensureNamespace creates the namespace if it doesn't exist
func (m *manager) ensureNamespace(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Infof("ensuring namespace %s", installerInputs.Namespace)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: installerInputs.Namespace,
			Labels: map[string]string{
				"aro-cluster-uuid":  installerInputs.ClusterUUID,
				"installer-backend": "aksjob",
			},
		},
	}

	_, err := m.client.CoreV1().Namespaces().Get(ctx, installerInputs.Namespace, metav1.GetOptions{})
	if err == nil {
		m.log.Info("namespace already exists")
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	_, err = m.client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	m.log.Info("namespace created")
	return nil
}

// ensureServiceAccount creates the service account with workload identity annotations
func (m *manager) ensureServiceAccount(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("ensuring service account")

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "installer",
			Namespace:   installerInputs.Namespace,
			Annotations: map[string]string{
				// TODO: Get installer identity client ID from environment/config
				// "azure.workload.identity/client-id": "<installer-identity-client-id>",
			},
			Labels: map[string]string{
				"azure.workload.identity/use": "true",
				"execution-id":                installerInputs.ExecutionID,
			},
		},
	}

	_, err := m.client.CoreV1().ServiceAccounts(installerInputs.Namespace).Get(ctx, "installer", metav1.GetOptions{})
	if err == nil {
		m.log.Info("service account already exists")
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check service account: %w", err)
	}

	_, err = m.client.CoreV1().ServiceAccounts(installerInputs.Namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	m.log.Info("service account created")
	return nil
}

// ensureSecrets creates the secrets needed by the installer
func (m *manager) ensureSecrets(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("ensuring installer secrets")

	// Create main installer inputs secret
	secretData := map[string][]byte{
		"99_aro.json": installerInputs.ClusterJSON,
		"99_sub.json": installerInputs.SubscriptionJSON,
	}

	// Add service principal credentials if this is an SP cluster
	if len(installerInputs.ServicePrincipalJSON) > 0 {
		secretData["osServicePrincipal.json"] = installerInputs.ServicePrincipalJSON
	}

	// Add proxy certificates for development mode
	if installerInputs.IsDevelopmentMode {
		if len(installerInputs.ProxyCert) > 0 {
			secretData["proxy.crt"] = installerInputs.ProxyCert
		}
		if len(installerInputs.ProxyClientCert) > 0 {
			secretData["proxy-client.crt"] = installerInputs.ProxyClientCert
		}
		if len(installerInputs.ProxyClientKey) > 0 {
			secretData["proxy-client.key"] = installerInputs.ProxyClientKey
		}
	}

	installerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installer-inputs",
			Namespace: installerInputs.Namespace,
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	err := m.createOrUpdateSecret(ctx, installerSecret)
	if err != nil {
		return fmt.Errorf("failed to create installer-inputs secret: %w", err)
	}

	// Create bound SA signing key secret for managed identity clusters
	if len(installerInputs.BoundSASigningKey) > 0 {
		boundKeySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bound-sa-signing-key",
				Namespace: installerInputs.Namespace,
			},
			Data: map[string][]byte{
				"bound-service-account-signing-key.key": installerInputs.BoundSASigningKey,
			},
			Type: corev1.SecretTypeOpaque,
		}

		err = m.createOrUpdateSecret(ctx, boundKeySecret)
		if err != nil {
			return fmt.Errorf("failed to create bound-sa-signing-key secret: %w", err)
		}
	}

	// Create secrets for custom manifests
	if len(installerInputs.CustomManifests) > 0 {
		manifestsSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom-manifests",
				Namespace: installerInputs.Namespace,
			},
			Data: installerInputs.CustomManifests,
			Type: corev1.SecretTypeOpaque,
		}

		err = m.createOrUpdateSecret(ctx, manifestsSecret)
		if err != nil {
			return fmt.Errorf("failed to create custom-manifests secret: %w", err)
		}
	}

	m.log.Info("secrets created")
	return nil
}

// createOrUpdateSecret creates or updates a secret
func (m *manager) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	_, err := m.client.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err == nil {
		// Secret exists, update it
		_, err = m.client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	}
	if !errors.IsNotFound(err) {
		return err
	}

	// Secret doesn't exist, create it
	_, err = m.client.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

// ensureJob creates the installer Job
func (m *manager) ensureJob(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("ensuring installer Job")

	// Check if Job already exists
	existingJob, err := m.client.BatchV1().Jobs(installerInputs.Namespace).Get(ctx, "installer", metav1.GetOptions{})
	if err == nil {
		// Job exists - verify it matches our execution ID
		if existingJob.Labels["execution-id"] == installerInputs.ExecutionID {
			m.log.Info("reattaching to existing job")
			return nil
		}
		return fmt.Errorf("job exists with different execution ID - possible collision")
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check job: %w", err)
	}

	// Build environment variables
	envVars := []corev1.EnvVar{}
	for key, value := range installerInputs.EnvironmentVariables {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Build volume mounts and volumes
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "installer-inputs",
			MountPath: "/var/run/installer-inputs",
			ReadOnly:  true,
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "installer-inputs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "installer-inputs",
				},
			},
		},
	}

	// Add bound SA signing key volume for managed identity clusters
	if len(installerInputs.BoundSASigningKey) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "bound-sa-signing-key",
			MountPath: "/var/run/secrets/openshift/bound-sa-signing-key",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "bound-sa-signing-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "bound-sa-signing-key",
				},
			},
		})
	}

	// Add custom manifests volume if present
	if len(installerInputs.CustomManifests) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "custom-manifests",
			MountPath: "/var/run/installer-custom-manifests",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "custom-manifests",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "custom-manifests",
				},
			},
		})
	}

	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installer",
			Namespace: installerInputs.Namespace,
			Labels: map[string]string{
				"aro-cluster-uuid":               installerInputs.ClusterUUID,
				"execution-id":                   installerInputs.ExecutionID,
				"installer-backend":              "aksjob",
				"kubernetes.azure.com/managedby": "sub_" + installerInputs.SubscriptionID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          pointerutils.ToPtr(int32(0)),    // No retries
			ActiveDeadlineSeconds: pointerutils.ToPtr(int64(3600)), // 60 min timeout
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"aro-cluster-uuid":               installerInputs.ClusterUUID,
						"execution-id":                   installerInputs.ExecutionID,
						"kubernetes.azure.com/managedby": "sub_" + installerInputs.SubscriptionID,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "installer",
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "installer",
							Image: installerInputs.InstallerPullspec,
							Command: []string{
								"/bin/bash",
								"-c",
								"/bin/openshift-install create manifests && /bin/openshift-install create cluster",
							},
							Env:          envVars,
							VolumeMounts: volumeMounts,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("4"),
									corev1.ResourceMemory: resource.MustParse("8Gi"),
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	_, err = m.client.BatchV1().Jobs(installerInputs.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	m.log.Info("job created successfully")
	return nil
}

// jobCompleted checks if the Job has completed (success or failure)
func (m *manager) jobCompleted(ctx context.Context, installerInputs *inputs.InstallerInputs) (bool, error) {
	job, err := m.client.BatchV1().Jobs(installerInputs.Namespace).Get(ctx, "installer", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			m.log.Debug("job not found yet")
			return false, nil
		}
		return false, fmt.Errorf("failed to get job: %w", err)
	}

	// Check for successful completion
	if job.Status.Succeeded > 0 {
		m.log.Info("job completed successfully")
		return true, nil
	}

	// Check for failure
	if job.Status.Failed > 0 {
		m.log.Error("job failed, retrieving logs for error analysis")

		// Get logs to parse for specific error
		logs, err := m.getStoredLogs(ctx, installerInputs)
		if err != nil {
			m.log.Warnf("failed to retrieve logs for error parsing: %v", err)
			logs = ""
		}

		// Parse logs and return structured error (will implement parsing next)
		return true, m.parseInstallationError(logs)
	}

	// Check for timeout
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobFailed {
			if cond.Reason == "DeadlineExceeded" {
				m.log.Error("job timed out")
				return true, fmt.Errorf("installer job timed out after %d seconds", *job.Spec.ActiveDeadlineSeconds)
			}
			if cond.Status == corev1.ConditionTrue {
				m.log.Errorf("job failed: %s - %s", cond.Reason, cond.Message)

				// Get logs for error parsing
				logs, err := m.getStoredLogs(ctx, installerInputs)
				if err != nil {
					m.log.Warnf("failed to retrieve logs: %v", err)
					logs = ""
				}

				return true, m.parseInstallationError(logs)
			}
		}
	}

	// Still running
	m.log.Debugf("job still running: active=%d, succeeded=%d, failed=%d",
		job.Status.Active, job.Status.Succeeded, job.Status.Failed)
	return false, nil
}

// parseInstallationError analyzes installation logs to return a structured error
// Reuses Hive's failure parsing logic from pkg/hive/failure
func (m *manager) parseInstallationError(logs string) error {
	if logs == "" {
		m.log.Warn("no logs available for error parsing")
		return &api.CloudError{
			StatusCode: http.StatusInternalServerError,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInternalServerError,
				Message: "Deployment failed.",
			},
		}
	}

	m.log.Infof("parsing %d bytes of installer logs for known error patterns", len(logs))
	return parseInstallationFailure(logs)
}

// collectLogs retrieves and stores the Job logs in a ConfigMap
func (m *manager) collectLogs(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("collecting installer logs")

	logs, err := m.getLogsFromPod(ctx, installerInputs.Namespace)
	if err != nil {
		m.log.Warnf("failed to get logs from pod: %v", err)
		return nil // Non-fatal
	}

	if logs == "" {
		m.log.Warn("no logs retrieved from installer pod")
		return nil
	}

	// Store logs in ConfigMap (like Hive stores in ClusterProvision.Spec.InstallLog)
	err = m.storeLogsInConfigMap(ctx, installerInputs, logs)
	if err != nil {
		m.log.Warnf("failed to store logs in ConfigMap: %v", err)
		return nil // Non-fatal
	}

	m.log.Infof("stored installer logs (%d bytes) in ConfigMap", len(logs))
	return nil
}

// getLogsFromPod retrieves logs directly from the installer pod
func (m *manager) getLogsFromPod(ctx context.Context, namespace string) (string, error) {
	// Find pods for the job
	pods, err := m.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job-name=installer",
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for installer job")
	}

	// Get logs from the first pod
	pod := pods.Items[0]
	req := m.client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "installer",
	})

	logStream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to stream logs: %w", err)
	}
	defer logStream.Close()

	// Read logs
	buf := new(strings.Builder)
	_, err = io.Copy(buf, logStream)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// storeLogsInConfigMap stores installer logs in a ConfigMap for retrieval
func (m *manager) storeLogsInConfigMap(ctx context.Context, installerInputs *inputs.InstallerInputs, logs string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installer-logs",
			Namespace: installerInputs.Namespace,
			Labels: map[string]string{
				"execution-id": installerInputs.ExecutionID,
				"cluster-uuid": installerInputs.ClusterUUID,
			},
		},
		Data: map[string]string{
			"install.log": logs,
		},
	}

	// Try to create the ConfigMap
	_, err := m.client.CoreV1().ConfigMaps(installerInputs.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// If it already exists, update it
	if errors.IsAlreadyExists(err) {
		_, err = m.client.CoreV1().ConfigMaps(installerInputs.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update ConfigMap: %w", err)
		}
	}

	return nil
}

// getStoredLogs retrieves logs from ConfigMap or pod
func (m *manager) getStoredLogs(ctx context.Context, installerInputs *inputs.InstallerInputs) (string, error) {
	// First try to get logs from ConfigMap (stored by collectLogs)
	cm, err := m.client.CoreV1().ConfigMaps(installerInputs.Namespace).Get(ctx, "installer-logs", metav1.GetOptions{})
	if err == nil {
		if logs, ok := cm.Data["install.log"]; ok {
			return logs, nil
		}
	}

	// Fallback to getting logs directly from pod
	m.log.Debug("ConfigMap not found, retrieving logs from pod")
	return m.getLogsFromPod(ctx, installerInputs.Namespace)
}
