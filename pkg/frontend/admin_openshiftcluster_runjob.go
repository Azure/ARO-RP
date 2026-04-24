package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	runJobDefaultNamespace = "openshift-azure-operator"
	kubeJobResource        = "Job.batch" // dot-separated Kind.Group format for non-core API group

	// defaultJobPollInterval is the delay between job status polls; pass 0 in tests.
	defaultJobPollInterval = 2 * time.Second

	jobResultSucceeded = "succeeded"
	jobResultFailed    = "failed"
	jobResultPending   = "pending"
)

func (f *frontend) postAdminOpenShiftClusterRunJob(w http.ResponseWriter, r *http.Request) {
	f.adminStreamAction(w, r, f._postAdminOpenShiftClusterRunJob)
}

func (f *frontend) _postAdminOpenShiftClusterRunJob(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	body := ctx.Value(middleware.ContextKeyBody).([]byte)
	job, err := parseAndValidateJob(body)
	if err != nil {
		return err
	}

	k, resourceID, err := f.fetchClusterKubeActions(ctx, r, log)
	if err != nil {
		return err
	}

	log = log.WithField("resourceID", resourceID)
	opCtx, opCancel := context.WithTimeout(ctx, adminActionStreamTimeout)
	// runJobStream installs its own panic recovery via utilrecover.Panic.
	go func() {
		defer opCancel()
		runJobStream(opCtx, log, k, job, writer, defaultJobPollInterval)
	}()
	return nil
}

func parseAndValidateJob(body []byte) (*batchv1.Job, error) {
	if len(body) == 0 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			"The request body must not be empty.")
	}

	var job batchv1.Job
	if err := json.Unmarshal(body, &job); err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			"Failed to parse request body.")
	}

	if job.Kind != "" && job.Kind != "Job" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("Expected kind 'Job', got '%s'.", job.Kind))
	}

	// Clear server-managed fields; API server rejects creates with resourceVersion set.
	job.UID = ""
	job.ResourceVersion = ""
	job.CreationTimestamp = metav1.Time{}
	job.DeletionTimestamp = nil
	job.ManagedFields = nil
	job.Generation = 0
	job.Status = batchv1.JobStatus{}
	job.Finalizers = nil
	job.OwnerReferences = nil

	if job.Name == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"The provided Job manifest must have a non-empty metadata.name.")
	}

	if !rxKubernetesString.MatchString(job.Name) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided Job metadata.name '%s' is invalid.", job.Name))
	}

	if job.Namespace == "" {
		job.Namespace = runJobDefaultNamespace
	} else {
		if !rxKubernetesString.MatchString(job.Namespace) {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
				fmt.Sprintf("The provided Job metadata.namespace '%s' is invalid.", job.Namespace))
		}
		if !utilnamespace.IsOpenShiftNamespace(job.Namespace) {
			return nil, api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "",
				fmt.Sprintf("Access to the provided namespace '%s' is forbidden.", job.Namespace))
		}
	}

	// Only single-pod jobs are supported; multi-pod log streaming is not implemented.
	if job.Spec.Parallelism != nil && *job.Spec.Parallelism > 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"Jobs with spec.parallelism > 1 are not implemented.")
	}
	if job.Spec.Completions != nil && *job.Spec.Completions > 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"Jobs with spec.completions > 1 are not implemented.")
	}

	// Retries create additional pods whose logs would not be captured.
	if job.Spec.BackoffLimit != nil && *job.Spec.BackoffLimit != 0 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"Jobs with spec.backoffLimit != 0 are not supported; set it to 0 or omit it.")
	}
	job.Spec.BackoffLimit = pointerutils.ToPtr(int32(0))

	// restartPolicy must be Never or OnFailure; "Always" would cause the pod to restart indefinitely.
	if rp := job.Spec.Template.Spec.RestartPolicy; rp != corev1.RestartPolicyNever && rp != corev1.RestartPolicyOnFailure {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The Job pod template restartPolicy must be Never or OnFailure, got %q.", rp))
	}

	// Exactly one container required: log streaming needs an unambiguous target.
	if len(job.Spec.Template.Spec.Containers) != 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The Job pod template must define exactly one container, got %d.", len(job.Spec.Template.Spec.Containers)))
	}

	containerName := job.Spec.Template.Spec.Containers[0].Name
	if containerName == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"The Job pod template must specify a non-empty container name.")
	}
	if !rxKubernetesString.MatchString(containerName) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The Job pod template container name '%s' is invalid.", containerName))
	}

	// Truncate to 57 chars so the final name (+ "-XXXXX" suffix) fits the 63-char DNS label limit.
	const maxJobBaseName = 57
	if len(job.Name) > maxJobBaseName {
		job.Name = strings.TrimRight(job.Name[:maxJobBaseName], "-.")
	}
	if job.Name == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"The provided Job metadata.name reduces to empty after truncation.")
	}
	job.Name = job.Name + "-" + utilrand.String(5) // 5 random chars: 57 base + 1 dash + 5 = 63-char DNS label limit

	return &job, nil
}

// runJobStream creates a Job, streams its pod logs, then deletes it; pass delay=0 in tests.
func runJobStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, job *batchv1.Job, w io.WriteCloser, delay time.Duration) {
	defer utilrecover.Panic(log)
	defer w.Close()

	namespace := job.Namespace
	jobName := job.Name
	log = log.WithFields(logrus.Fields{"namespace": namespace, "jobName": jobName})
	log.Info("admin runjob")

	// Watch before create to avoid missing the pod-created event.
	podTemplate := &unstructured.Unstructured{}
	podTemplate.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	podTemplate.SetNamespace(namespace)
	podTemplate.SetLabels(map[string]string{"batch.kubernetes.io/job-name": jobName})

	watcher, err := k.KubeWatch(ctx, podTemplate, "batch.kubernetes.io/job-name")
	if err != nil {
		log.WithError(err).Warn("error setting up pod watch")
		fmt.Fprintf(w, "Error setting up pod watch: %v\n", err)
		return
	}
	stopWatcher := sync.OnceFunc(watcher.Stop)
	defer stopWatcher()

	// Write errors are intentionally ignored; the pipe reader may close early on client disconnect.
	fmt.Fprintf(w, "Creating job %s in %s...\n", jobName, namespace)

	unstrMap, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		log.WithError(err).Warn("failed to prepare job manifest")
		fmt.Fprintf(w, "Failed to prepare job manifest: %v\n", err)
		return
	}
	un := &unstructured.Unstructured{Object: unstrMap}
	un.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})

	if err := k.KubeCreateOrUpdate(ctx, un); err != nil {
		log.WithError(err).Warn("failed to create job")
		fmt.Fprintf(w, "Failed to create job: %v\n", err)
		// No cleanupJob: a rejected create leaves no resource; a transient-accepted-lost-response job will be GC'd by Kubernetes TTL.
		return
	}

	fmt.Fprintf(w, "Waiting for pod...\n")
	podName, err := waitForJobPod(ctx, log, watcher)
	stopWatcher() // release watch connection promptly; no longer needed after pod is found
	if err != nil {
		// Two ctx.Err() checks are intentional: the first selects log level (Warn vs Error);
		// the second suppresses stream output on cancellation. cleanupJob uses context.Background()
		// so it runs regardless of context state.
		if ctx.Err() != nil {
			log.WithError(err).Warn("wait for job pod cancelled")
		} else {
			log.WithError(err).WithField("stage", "waitForPod").Error("error waiting for job pod")
		}
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).WithField("stage", "cleanup").Error("job cleanup failed")
		}
		if ctx.Err() != nil {
			return
		}
		fmt.Fprintf(w, "Error waiting for pod: %v\n", err)
		return
	}

	if ctx.Err() != nil {
		log.Info("job cancelled before log streaming")
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).WithField("stage", "cleanup").Error("job cleanup failed")
		}
		return
	}

	fmt.Fprintf(w, "Pod %s assigned, streaming logs...\n", podName)

	// parseAndValidateJob guarantees at least one container.
	containerName := job.Spec.Template.Spec.Containers[0].Name
	if err := k.KubeFollowPodLogs(ctx, namespace, podName, containerName, newLimitedWriter(w, "pod logs", log)); err != nil {
		if ctx.Err() != nil {
			log.WithError(err).Warn("job pod log streaming cancelled")
		} else {
			log.WithError(err).Warn("job pod log streaming error")
			fmt.Fprintf(w, "Log streaming error: %v\n", err)
		}
	}

	if ctx.Err() != nil {
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).WithField("stage", "cleanup").Error("job cleanup failed")
		}
		return
	}

	// Wait for terminal state in case log streaming ended early (e.g. truncation).
	// Loop runs until job reaches a terminal condition or opCtx (adminActionStreamTimeout) fires.
	result := waitForJobTerminal(ctx, log, k, namespace, jobName, delay)
	if ctx.Err() != nil {
		log.Info("job polling cancelled by context")
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).WithField("stage", "cleanup").Error("job cleanup failed")
		}
		return
	}
	switch result {
	case jobResultSucceeded:
		log.WithField("jobResult", result).Info("job reached terminal state")
		fmt.Fprintf(w, "Job succeeded.\n")
	case jobResultFailed:
		log.WithField("jobResult", result).Warn("job reached terminal state")
		fmt.Fprintf(w, "Job failed.\n")
	default:
		log.WithField("jobResult", result).Warn("job polling exited without terminal state")
		// Raw Kubernetes status written intentionally: this is an admin-only diagnostic channel.
		fmt.Fprintf(w, "Job polling exhausted (last status: %s)\n", result)
	}

	if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
		log.WithError(cleanupErr).WithField("stage", "cleanup").Error("job cleanup failed")
		fmt.Fprintf(w, "Cleanup failed: %v\n", cleanupErr)
	} else {
		fmt.Fprintf(w, "Cleanup complete.\n")
	}
}

// waitForJobPod reads from watcher until a pod reaches a non-Pending phase and returns its name.
func waitForJobPod(ctx context.Context, log *logrus.Entry, watcher watch.Interface) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				if ctx.Err() != nil {
					return "", ctx.Err()
				}
				return "", errors.New("pod watch channel closed unexpectedly")
			}
			if event.Type == watch.Error {
				if status, ok := event.Object.(*metav1.Status); ok {
					return "", fmt.Errorf("pod watch error: %s (reason: %s, code: %d)", status.Message, status.Reason, status.Code)
				}
				return "", fmt.Errorf("pod watch error: unexpected object type %T", event.Object)
			}
			// Skip Deleted/Bookmark events; only Added/Modified indicate pod state updates.
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			pod, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				log.Warn("unexpected pod event object type, skipping")
				continue
			}
			name := pod.GetName()
			phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
			// PodPending is deliberately not surfaced; the caller's opCtx (30-minute timeout) is the only bound for stuck-pending pods.
			switch corev1.PodPhase(phase) {
			case corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed:
				return name, nil
			}
		}
	}
}

// jobResult returns "succeeded", "failed", or "pending" based on the Job's conditions.
func jobResult(ctx context.Context, k adminactions.KubeActions, namespace, jobName string) (string, error) {
	data, err := k.KubeGet(ctx, kubeJobResource, namespace, jobName)
	if err != nil {
		return "", fmt.Errorf("fetching job status: %w", err)
	}
	var job batchv1.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("parsing job status: %w", err)
	}
	for _, cond := range job.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		// JobSuspended and FailureTarget are intentionally treated as pending; backoffLimit=0 makes FailureTarget unreachable.
		switch cond.Type {
		case batchv1.JobComplete:
			return jobResultSucceeded, nil
		case batchv1.JobFailed:
			return jobResultFailed, nil
		}
	}
	return jobResultPending, nil
}

// waitForJobTerminal polls until the Job reaches a terminal state; exits after 10 consecutive errors.
func waitForJobTerminal(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, namespace, jobName string, delay time.Duration) string {
	const maxConsecutiveErrors = 10
	consecutiveErrors := 0
	var lastResult string
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return lastResult
		case <-timer.C:
		}
		result, err := jobResult(ctx, k, namespace, jobName)
		if err != nil {
			lastResult = err.Error()
			consecutiveErrors++
			log.WithError(err).Warn("job status poll failed")
			if consecutiveErrors >= maxConsecutiveErrors {
				return lastResult
			}
			timer.Reset(delay)
			continue
		}
		consecutiveErrors = 0
		lastResult = result
		switch result {
		case jobResultSucceeded, jobResultFailed:
			return result
		}
		timer.Reset(delay)
	}
}

// cleanupJob deletes the Job with a fresh context (request cancellation must not prevent cleanup).
func cleanupJob(k adminactions.KubeActions, namespace, jobName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), adminActionCleanupTimeout)
	defer cancel()
	foreground := metav1.DeletePropagationForeground
	return retry.OnError(kubeRetryBackoff, func(err error) bool {
		return kerrors.IsInternalError(err) || kerrors.IsServerTimeout(err) || kerrors.IsServiceUnavailable(err)
	}, func() error {
		err := k.KubeDelete(ctx, kubeJobResource, namespace, jobName, false, &foreground)
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	})
}
