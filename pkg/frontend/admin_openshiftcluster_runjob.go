package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
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
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	runJobDefaultNamespace = "openshift-azure-operator"
	kubeJobResource        = "Job.batch" // dot-separated groupKind for non-core API group

	jobResultSucceeded = "succeeded"
	jobResultFailed    = "failed"
	jobResultPending   = "pending"
)

func (f *frontend) postAdminOpenShiftClusterRunJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	defer reader.Close()
	err := f._postAdminOpenShiftClusterRunJob(ctx, r, log, writer)
	if err != nil {
		_ = writer.CloseWithError(err)
	}
	var header http.Header
	if err == nil {
		header = http.Header{"Content-Type": []string{"text/plain"}}
	}
	f.streamResponder.AdminReplyStream(log, w, header, reader, err)
}

func (f *frontend) _postAdminOpenShiftClusterRunJob(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	job, err := parseAndValidateJob(body)
	if err != nil {
		return err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return fmt.Errorf("fetching cluster document: %w", err)
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return fmt.Errorf("creating kube actions: %w", err)
	}

	log = log.WithField("resourceID", resourceID)
	opCtx, opCancel := context.WithCancel(ctx)
	go func() {
		defer opCancel()
		runJobStream(opCtx, log, k, job, writer, defaultJobRetryDelay)
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
			fmt.Sprintf("Failed to parse request body: %v", err))
	}

	if job.Kind != "Job" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("Expected kind 'Job', got '%s'.", job.Kind))
	}

	// Clear server-managed fields. The API server rejects creates with
	// resourceVersion set, and the others have no meaning in a submitted manifest.
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

	// Retries would create additional pods whose logs would not be captured
	// by this single-pod streaming path.
	if job.Spec.BackoffLimit != nil && *job.Spec.BackoffLimit != 0 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"Jobs with spec.backoffLimit != 0 are not supported; set it to 0 or omit it.")
	}
	zero := int32(0)
	job.Spec.BackoffLimit = &zero

	// Exactly one container is required so that log streaming targets an
	// unambiguous container rather than relying on Kubernetes' default selection,
	// which fails for multi-container pods.
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

	// Kubernetes Job names must fit within 57 characters so that the final
	// name (with our 6-character "-XXXXX" suffix) stays within the 63-char
	// DNS label limit required for pod name validity.
	const maxJobBaseName = 57
	if len(job.Name) > maxJobBaseName {
		job.Name = job.Name[:maxJobBaseName]
	}
	job.Name = job.Name + "-" + utilrand.String(5) // 5 random chars: 57 base + 1 dash + 5 = 63-char DNS label limit

	return &job, nil
}

// runJobStream creates a Job, streams its pod logs, then deletes it. Pass a nopWriteCloser
// when calling from etcdAnalysisStream to prevent premature close; pass delay=0 in tests.
func runJobStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, job *batchv1.Job, w io.WriteCloser, delay time.Duration) {
	defer utilrecover.Panic(log)
	defer w.Close()

	namespace := job.Namespace
	jobName := job.Name

	// Watch before create to avoid missing the pod-created event.
	podTemplate := &unstructured.Unstructured{}
	podTemplate.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	podTemplate.SetNamespace(namespace)
	podTemplate.SetLabels(map[string]string{"batch.kubernetes.io/job-name": jobName})

	watcher, err := k.KubeWatch(ctx, podTemplate, "batch.kubernetes.io/job-name")
	if err != nil {
		log.WithError(err).Error("error setting up pod watch")
		fmt.Fprintf(w, "Error setting up pod watch: %v", err)
		return
	}
	stopWatcher := sync.OnceFunc(watcher.Stop)
	defer stopWatcher()

	fmt.Fprintf(w, "Creating job %s in %s...\n", jobName, namespace)

	unstrMap, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		fmt.Fprintf(w, "Failed to prepare job manifest: %v", err)
		return
	}
	un := &unstructured.Unstructured{Object: unstrMap}
	un.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})

	if err := k.KubeCreateOrUpdate(ctx, un); err != nil {
		fmt.Fprintf(w, "Failed to create job: %v", err)
		return
	}

	fmt.Fprintf(w, "Waiting for pod...\n")
	podName, err := waitForJobPod(ctx, watcher)
	stopWatcher() // release watch connection promptly; no longer needed after pod is found
	if err != nil {
		log.WithError(err).Error("error waiting for job pod")
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).Error("job cleanup failed")
			fmt.Fprintf(w, "Error waiting for pod: %v\nCleanup failed: %v", err, cleanupErr)
		} else {
			fmt.Fprintf(w, "Error waiting for pod: %v", err)
		}
		return
	}

	fmt.Fprintf(w, "Pod %s assigned, streaming logs...\n", podName)

	containerName := job.Spec.Template.Spec.Containers[0].Name
	if err := k.KubeFollowPodLogs(ctx, namespace, podName, containerName, newLimitedWriter(w, "pod logs")); err != nil && ctx.Err() == nil {
		log.WithError(err).Warn("job pod log streaming error")
		fmt.Fprintf(w, "Log streaming error: %v\n", err)
	}

	if ctx.Err() != nil {
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			log.WithError(cleanupErr).Error("job cleanup failed")
			fmt.Fprintf(w, "Request cancelled.\nCleanup failed: %v", cleanupErr)
		} else {
			fmt.Fprintf(w, "Request cancelled.")
		}
		return
	}

	// Wait for terminal state in case log streaming ended early (e.g. truncation).
	result := waitForJobTerminal(ctx, log, k, namespace, jobName, delay)
	if result == "" {
		result = "cancelled"
	}
	log.WithField("jobResult", result).Info("job reached terminal state")
	switch result {
	case jobResultSucceeded:
		fmt.Fprintf(w, "Job succeeded.\n")
	case jobResultFailed:
		fmt.Fprintf(w, "Job failed.\n")
	default:
		fmt.Fprintf(w, "Job status: %s\n", result)
	}

	if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
		log.WithError(cleanupErr).Error("job cleanup failed")
		fmt.Fprintf(w, "Cleanup failed: %v", cleanupErr)
	} else {
		fmt.Fprintf(w, "Cleanup complete.")
	}
}

// waitForJobPod reads from watcher until a pod reaches a non-Pending phase and returns its name.
func waitForJobPod(ctx context.Context, watcher watch.Interface) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("pod watch channel closed unexpectedly")
			}
			if event.Type == watch.Error {
				if status, ok := event.Object.(*metav1.Status); ok {
					return "", fmt.Errorf("pod watch error: %s (reason: %s, code: %d)", status.Message, status.Reason, status.Code)
				}
				return "", fmt.Errorf("pod watch error: unexpected object type %T", event.Object)
			}
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			pod, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			name := pod.GetName()
			phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
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
		return "", fmt.Errorf("could not fetch job status: %w", err)
	}
	var job batchv1.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return "", fmt.Errorf("could not parse job status: %w", err)
	}
	for _, cond := range job.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
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

// cleanupJob deletes the Job using a fresh context; retries on transient errors.
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
