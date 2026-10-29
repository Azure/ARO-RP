package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStatus returns the current status of an installer Job
func (m *manager) GetStatus(ctx context.Context, namespace, jobName string) (*JobStatus, error) {
	m.log.Infof("getting status for Job %s/%s", namespace, jobName)

	job, err := m.client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &JobStatus{
				Phase:   JobPhaseUnknown,
				Message: "Job not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	status := &JobStatus{}

	// Determine phase
	if job.Status.Succeeded > 0 {
		status.Phase = JobPhaseSucceeded
		status.Message = "Installation completed successfully"
	} else if job.Status.Failed > 0 {
		status.Phase = JobPhaseFailed
		status.Message = "Installation failed"
	} else if job.Status.Active > 0 {
		status.Phase = JobPhaseRunning
		status.Message = "Installation in progress"
	} else {
		status.Phase = JobPhasePending
		status.Message = "Installation pending"
	}

	// Check for timeout
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobFailed && cond.Reason == "DeadlineExceeded" {
			status.Phase = JobPhaseTimedOut
			status.Message = fmt.Sprintf("Installation timed out: %s", cond.Message)
		}
	}

	// Get timing information
	if job.Status.StartTime != nil {
		startTime := job.Status.StartTime.Format("2006-01-02T15:04:05Z")
		status.StartTime = &startTime
	}
	if job.Status.CompletionTime != nil {
		completionTime := job.Status.CompletionTime.Format("2006-01-02T15:04:05Z")
		status.CompletionTime = &completionTime
	}

	// Try to get exit code from pod
	pods, err := m.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err == nil && len(pods.Items) > 0 {
		pod := pods.Items[0]
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == "installer" && containerStatus.State.Terminated != nil {
				status.ExitCode = &containerStatus.State.Terminated.ExitCode
				if containerStatus.State.Terminated.Message != "" {
					status.Message = containerStatus.State.Terminated.Message
				}
			}
		}
	}

	return status, nil
}

// GetLogs retrieves logs from the installer Job
func (m *manager) GetLogs(ctx context.Context, namespace, jobName string) (string, error) {
	m.log.Infof("getting logs for Job %s/%s", namespace, jobName)

	// Find pods for the job
	pods, err := m.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s/%s", namespace, jobName)
	}

	// Get logs from the first pod
	pod := pods.Items[0]
	req := m.client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "installer",
	})

	logs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to stream logs: %w", err)
	}
	defer logs.Close()

	// Read logs
	buf := new(strings.Builder)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}
