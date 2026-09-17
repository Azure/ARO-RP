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

	"github.com/sirupsen/logrus"

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

const (
	maxInstallerLogBytes = 1024 * 1024
)

type executionResourceNames struct {
	inputsSecret    string
	boundKeySecret  string
	manifestsSecret string
	pullSecret      string
}

func resourceNames(jobName string) executionResourceNames {
	return executionResourceNames{
		inputsSecret:    jobName + "-inputs",
		boundKeySecret:  jobName + "-bound-key",
		manifestsSecret: jobName + "-manifests",
		pullSecret:      jobName + "-pull",
	}
}

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
	}

	_, err := steps.Run(ctx, m.log, 10*time.Second, s, nil, "")
	if err == nil {
		_, err = steps.Run(ctx, m.log, 10*time.Second, []steps.Step{
			steps.Condition(func(ctx context.Context) (bool, error) {
				return m.jobCompleted(ctx, installerInputs)
			}, installerInputs.Timeout, true),
		}, nil, "")
	}

	if logErr := m.collectLogs(ctx, installerInputs); logErr != nil {
		m.log.WithError(logErr).Warn("failed to collect installer logs")
	}

	return err
}

// ensureNamespace verifies the SVC deployment has provisioned the shared
// installer namespace.
func (m *manager) ensureNamespace(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	_, err := m.client.CoreV1().Namespaces().Get(ctx, installerInputs.Namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("installer namespace %s is not available: %w", installerInputs.Namespace, err)
	}
	return nil
}

// ensureServiceAccount verifies the ServiceAccount bound to the pre-provisioned
// installer workload identity.
func (m *manager) ensureServiceAccount(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	serviceAccount, err := m.client.CoreV1().ServiceAccounts(installerInputs.Namespace).Get(ctx, ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("installer ServiceAccount %s/%s is not available: %w", installerInputs.Namespace, ServiceAccountName, err)
	}

	if installerInputs.InstallerIdentityClientID != "" &&
		serviceAccount.Annotations["azure.workload.identity/client-id"] != installerInputs.InstallerIdentityClientID {
		return fmt.Errorf("installer ServiceAccount %s/%s has unexpected workload identity client ID", installerInputs.Namespace, ServiceAccountName)
	}
	return nil
}

// ensureSecrets creates the secrets needed by the installer
func (m *manager) ensureSecrets(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("ensuring installer secrets")
	names := resourceNames(installerInputs.JobName)

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
			Name:      names.inputsSecret,
			Namespace: installerInputs.Namespace,
			Labels:    executionLabels(installerInputs),
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	err := m.createOrUpdateSecret(ctx, installerSecret)
	if err != nil {
		return fmt.Errorf("failed to create installer inputs secret: %w", err)
	}

	// Create bound SA signing key secret for managed identity clusters
	if len(installerInputs.BoundSASigningKey) > 0 {
		boundKeySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.boundKeySecret,
				Namespace: installerInputs.Namespace,
				Labels:    executionLabels(installerInputs),
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
				Name:      names.manifestsSecret,
				Namespace: installerInputs.Namespace,
				Labels:    executionLabels(installerInputs),
			},
			Data: installerInputs.CustomManifests,
			Type: corev1.SecretTypeOpaque,
		}

		err = m.createOrUpdateSecret(ctx, manifestsSecret)
		if err != nil {
			return fmt.Errorf("failed to create custom-manifests secret: %w", err)
		}
	}

	if len(installerInputs.PullSecretJSON) > 0 {
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.pullSecret,
				Namespace: installerInputs.Namespace,
				Labels:    executionLabels(installerInputs),
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: installerInputs.PullSecretJSON,
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		if err := m.createOrUpdateSecret(ctx, pullSecret); err != nil {
			return fmt.Errorf("failed to create installer pull secret: %w", err)
		}
	}

	m.log.Info("secrets created")
	return nil
}

// createOrUpdateSecret creates or updates a secret
func (m *manager) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	existing, err := m.client.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err == nil {
		secret.ResourceVersion = existing.ResourceVersion
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
	names := resourceNames(installerInputs.JobName)

	// Check if Job already exists
	existingJob, err := m.client.BatchV1().Jobs(installerInputs.Namespace).Get(ctx, installerInputs.JobName, metav1.GetOptions{})
	if err == nil {
		// Job exists - verify it matches our execution ID
		if existingJob.Labels["execution-id"] == installerInputs.ExecutionID &&
			existingJob.Labels["aro-cluster-uuid"] == installerInputs.ClusterUUID {
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
			MountPath: "/.azure/99_aro.json",
			SubPath:   "99_aro.json",
			ReadOnly:  true,
		},
		{
			Name:      "installer-inputs",
			MountPath: "/.azure/99_sub.json",
			SubPath:   "99_sub.json",
			ReadOnly:  true,
		},
		{
			Name:      "azure-workdir",
			MountPath: "/.azure",
		},
		{
			Name:      "output",
			MountPath: "/output",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "installer-inputs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: names.inputsSecret,
				},
			},
		},
		{
			Name: "azure-workdir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			Name: "output",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if len(installerInputs.ServicePrincipalJSON) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "installer-inputs",
			MountPath: "/.azure/osServicePrincipal.json",
			SubPath:   "osServicePrincipal.json",
			ReadOnly:  true,
		})
	}

	for key, value := range map[string][]byte{
		"proxy.crt":        installerInputs.ProxyCert,
		"proxy-client.crt": installerInputs.ProxyClientCert,
		"proxy-client.key": installerInputs.ProxyClientKey,
	} {
		if installerInputs.IsDevelopmentMode && len(value) > 0 {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "installer-inputs",
				MountPath: "/.azure/" + key,
				SubPath:   key,
				ReadOnly:  true,
			})
		}
	}

	// Add bound SA signing key volume for managed identity clusters
	if len(installerInputs.BoundSASigningKey) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "bound-sa-signing-key",
			MountPath: "/boundsasigningkey",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "bound-sa-signing-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: names.boundKeySecret,
				},
			},
		})
	}

	// Add custom manifests volume if present
	if len(installerInputs.CustomManifests) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "custom-manifests",
			MountPath: "/manifests",
			ReadOnly:  true,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "custom-manifests",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: names.manifestsSecret,
				},
			},
		})
	}

	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installerInputs.JobName,
			Namespace: installerInputs.Namespace,
			Labels:    executionLabels(installerInputs),
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          pointerutils.ToPtr(int32(0)),
			ActiveDeadlineSeconds: pointerutils.ToPtr(int64(installerInputs.Timeout.Seconds())),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"aro-cluster-uuid":               installerInputs.ClusterUUID,
						"execution-id":                   installerInputs.ExecutionID,
						"kubernetes.azure.com/managedby": "sub_" + installerInputs.SubscriptionID,
						"azure.workload.identity/use":    "true",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: names.pullSecret},
					},
					Containers: []corev1.Container{
						{
							Name:       "installer",
							Image:      installerInputs.InstallerPullspec,
							WorkingDir: "/.azure",
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
	job, err := m.client.BatchV1().Jobs(installerInputs.Namespace).Get(ctx, installerInputs.JobName, metav1.GetOptions{})
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

// collectLogs forwards the Job logs to the RP log pipeline before cleanup.
func (m *manager) collectLogs(ctx context.Context, installerInputs *inputs.InstallerInputs) error {
	m.log.Info("collecting installer logs")

	logs, err := m.getLogsFromPod(ctx, installerInputs.Namespace, installerInputs.JobName)
	if err != nil {
		m.log.Warnf("failed to get logs from pod: %v", err)
		return err
	}

	if logs == "" {
		m.log.Warn("no logs retrieved from installer pod")
		return nil
	}

	logs = truncateInstallerLogs(logs)
	for _, line := range strings.Split(strings.TrimSuffix(logs, "\n"), "\n") {
		m.log.WithFields(logrus.Fields{
			"clusterUUID": installerInputs.ClusterUUID,
			"executionID": installerInputs.ExecutionID,
		}).Infof("installer: %s", line)
	}

	return nil
}

func truncateInstallerLogs(logs string) string {
	if len(logs) <= maxInstallerLogBytes {
		return logs
	}

	const marker = "\n... installer logs truncated ...\n"
	remaining := maxInstallerLogBytes - len(marker)
	first := remaining / 2
	last := remaining - first
	return logs[:first] + marker + logs[len(logs)-last:]
}

// getLogsFromPod retrieves logs directly from the installer pod
func (m *manager) getLogsFromPod(ctx context.Context, namespace, jobName string) (string, error) {
	// Find pods for the job
	pods, err := m.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job-name=" + jobName,
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

// getStoredLogs retrieves logs from the installer pod.
func (m *manager) getStoredLogs(ctx context.Context, installerInputs *inputs.InstallerInputs) (string, error) {
	return m.getLogsFromPod(ctx, installerInputs.Namespace, installerInputs.JobName)
}

func executionLabels(installerInputs *inputs.InstallerInputs) map[string]string {
	return map[string]string{
		"aro-cluster-uuid":               installerInputs.ClusterUUID,
		"execution-id":                   installerInputs.ExecutionID,
		"installer-backend":              "aksjob",
		"kubernetes.azure.com/managedby": "sub_" + installerInputs.SubscriptionID,
	}
}
