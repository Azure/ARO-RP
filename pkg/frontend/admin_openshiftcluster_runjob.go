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

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilkubernetes "github.com/Azure/ARO-RP/pkg/util/kubernetes"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

const (
	runJobDefaultNamespace = "openshift-azure-operator"

	jobResultSucceeded = "succeeded"
	jobResultFailed    = "failed"
	jobResultPending   = "pending"
)

func (f *frontend) postAdminOpenShiftClusterRunJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
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
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	go runJobStream(ctx, k, job, writer)
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

	// runJobStream streams only one pod; retries would create additional pods
	// whose logs would not be captured. Clamp backoffLimit to 0 to prevent
	// silent retries from being lost.
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
	job.Name = job.Name + "-" + utilrand.String(5)

	return &job, nil
}

// runJobStream creates a Kubernetes Job on the cluster, streams the pod's logs back to w
// as they arrive, then deletes the Job regardless of outcome. It is called in a goroutine
// by the HTTP handler and may also be called directly by higher-level composed actions.
func runJobStream(ctx context.Context, k adminactions.KubeActions, job *batchv1.Job, w io.WriteCloser) {
	defer w.Close()

	namespace := job.Namespace
	jobName := job.Name

	// Establish the pod watch before creating the Job so we cannot miss the
	// pod-created event if the Job controller acts faster than our watch setup.
	podTemplate := &unstructured.Unstructured{}
	podTemplate.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	podTemplate.SetNamespace(namespace)
	podTemplate.SetLabels(map[string]string{"batch.kubernetes.io/job-name": jobName})

	watcher, err := k.KubeWatch(ctx, podTemplate, "batch.kubernetes.io/job-name")
	if err != nil {
		fmt.Fprintf(w, "Error setting up pod watch: %v\n", err)
		return
	}
	stopWatcher := sync.OnceFunc(watcher.Stop)
	defer stopWatcher()

	fmt.Fprintf(w, "Creating job %s in %s...\n", jobName, namespace)

	unstrMap, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		fmt.Fprintf(w, "Failed to prepare job manifest: %v\n", err)
		return
	}
	un := &unstructured.Unstructured{Object: unstrMap}
	un.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})

	if err := k.KubeCreateOrUpdate(ctx, un); err != nil {
		fmt.Fprintf(w, "Failed to create job: %v\n", err)
		return
	}

	fmt.Fprintf(w, "Waiting for pod...\n")
	podName, err := waitForJobPod(ctx, watcher)
	stopWatcher() // release watch connection promptly; no longer needed after pod is found
	if err != nil {
		fmt.Fprintf(w, "Error waiting for pod: %v\n", err)
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			fmt.Fprintf(w, "Cleanup failed: %v\n", cleanupErr)
		}
		return
	}

	fmt.Fprintf(w, "Pod %s assigned, streaming logs...\n", podName)

	// Use the explicit container name; an empty string fails when the pod has
	// multiple containers (e.g. injected sidecars).
	containerName := job.Spec.Template.Spec.Containers[0].Name
	if err := k.KubeFollowPodLogs(ctx, namespace, podName, containerName, newLimitedWriter(w, "pod logs")); err != nil && ctx.Err() == nil {
		fmt.Fprintf(w, "Log streaming error: %v\n", err)
	}

	if ctx.Err() != nil {
		fmt.Fprintf(w, "Request cancelled.\n")
		if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
			fmt.Fprintf(w, "Cleanup failed: %v\n", cleanupErr)
		}
		return
	}

	// waitForJobTerminal polls until the Job reaches a terminal state. This
	// handles the case where log streaming ended early (e.g. due to output
	// truncation) and the Job is still running; deleting a running Job
	// immediately would race with normal completion.
	switch result := waitForJobTerminal(ctx, k, namespace, jobName); result {
	case jobResultSucceeded:
		fmt.Fprintf(w, "Job succeeded.\n")
	case jobResultFailed:
		fmt.Fprintf(w, "Job failed.\n")
	default:
		fmt.Fprintf(w, "Job result: %s\n", result)
	}

	if cleanupErr := cleanupJob(k, namespace, jobName); cleanupErr != nil {
		fmt.Fprintf(w, "Cleanup failed: %v\n", cleanupErr)
	} else {
		fmt.Fprintf(w, "Cleanup complete.\n")
	}
}

// waitForJobPod reads from watcher until a pod reaches Running, Succeeded, or
// Failed phase, then returns the pod name.
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

// jobResult inspects the Job's conditions and returns "succeeded" (Complete=True),
// "failed" (Failed=True), or "pending" when no terminal condition is present
// (the Job controller may not have updated conditions yet).
func jobResult(ctx context.Context, k adminactions.KubeActions, namespace, jobName string) string {
	data, err := k.KubeGet(ctx, "Job.batch", namespace, jobName)
	if err != nil {
		return fmt.Sprintf("could not fetch job status: %v", err)
	}
	var job batchv1.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Sprintf("could not parse job status: %v", err)
	}
	for _, cond := range job.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case batchv1.JobComplete:
			return jobResultSucceeded
		case batchv1.JobFailed:
			return jobResultFailed
		}
	}
	return jobResultPending
}

// waitForJobTerminal polls jobResult every 2 seconds until the Job reaches a
// terminal state or the context is cancelled. Exits after 10 consecutive
// status-read errors to avoid looping indefinitely when the apiserver is down.
func waitForJobTerminal(ctx context.Context, k adminactions.KubeActions, namespace, jobName string) string {
	const maxConsecutiveErrors = 10
	consecutiveErrors := 0
	var result string
	for {
		result = jobResult(ctx, k, namespace, jobName)
		switch result {
		case jobResultSucceeded, jobResultFailed:
			return result
		case jobResultPending:
			consecutiveErrors = 0
		default:
			// jobResult returned an error string (KubeGet or JSON parse failure).
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return result
			}
		}
		if ctx.Err() != nil {
			return result
		}
		select {
		case <-ctx.Done():
			return result
		case <-time.After(2 * time.Second):
		}
	}
}

// cleanupJob deletes the Job with a fresh context so that cancellation of the
// caller's request context does not prevent cleanup. The delete is retried up
// to 3 times to handle transient apiserver flakiness, and not-found is treated
// as success since the job may have already been removed.
func cleanupJob(k adminactions.KubeActions, namespace, jobName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), adminActionCleanupTimeout)
	defer cancel()
	foreground := metav1.DeletePropagationForeground
	return utilkubernetes.Retry(ctx, 3, func() error {
		err := k.KubeDelete(ctx, "Job.batch", namespace, jobName, false, &foreground)
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	})
}
