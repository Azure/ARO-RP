package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	etcdAnalysisContainer = "etcdctl"
	etcdAnalysisImage     = "quay.io/redhat_emp1/octosql-etcd:latest"
	etcdSnapshotDir       = "/var/lib/etcd"

	// etcdAnalysisSAPrefix is the prefix for the per-request ServiceAccount
	// created in namespaceEtcds before running the analysis Job. A random
	// 8-character suffix is appended at runtime so that concurrent requests
	// do not collide on the same RBAC objects.
	etcdAnalysisSAPrefix = "etcd-analysis-privileged"
)

func (f *frontend) postAdminOpenShiftClusterEtcdAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	err := f._postAdminOpenShiftClusterEtcdAnalysis(ctx, r, log, writer)
	if err != nil {
		_ = writer.CloseWithError(err)
	}
	var header http.Header
	if err == nil {
		header = http.Header{"Content-Type": []string{"text/plain"}}
	}
	f.streamResponder.AdminReplyStream(log, w, header, reader, err)
}

func (f *frontend) _postAdminOpenShiftClusterEtcdAnalysis(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	nodeName := r.URL.Query().Get("nodeName")
	if nodeName == "" || !rxKubernetesString.MatchString(nodeName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided nodeName '%s' is invalid.", nodeName))
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

	go etcdAnalysisStream(ctx, log, k, nodeName, doc.OpenShiftCluster.Name, writer)
	return nil
}

// nopWriteCloser wraps an io.Writer with a no-op Close, allowing functions
// that close their writer (e.g. runJobStream) to be composed into a larger
// stream without prematurely closing the underlying writer.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// etcdAnalysisStream orchestrates the full etcd space analysis:
//  1. Creates an etcd snapshot via exec into the etcdctl container.
//  2. Defers a best-effort rm -f of the snapshot on all exit paths.
//  3. Creates a privileged ServiceAccount for the analysis Job (reusing the
//     createPrivilegedServiceAccount helper from fixEtcd).
//  4. Launches an analysis Job against the snapshot file.
//
// Technique sourced from:
// https://github.com/openshift/configuration-anomaly-detection/blob/main/pkg/investigations/etcddatabasequotalowspace/analysis.go
//
// It is called in a goroutine by the HTTP handler.
func etcdAnalysisStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, nodeName, clusterName string, w io.WriteCloser) {
	defer w.Close()

	podName := "etcd-" + nodeName
	// Use underscores (not dashes) and .snapshot extension. octosql parses
	// the FROM clause as SQL, so dashes in the path are misread as subtraction
	// operators. The etcdsnapshot plugin is registered only for ".snapshot"
	// files via file_extension_handlers.json in the container image.
	filename := fmt.Sprintf("etcd_analysis_%d.snapshot", time.Now().UnixNano())
	snapshotPath := etcdSnapshotDir + "/" + filename
	saName := etcdAnalysisSAPrefix + "-" + utilrand.String(8)
	saAccount := "system:serviceaccount:" + namespaceEtcds + ":" + saName

	// Always attempt to remove the snapshot file on exit, registered before
	// the exec so partial files from a failed save are also cleaned up.
	// A fresh context ensures cancellation does not prevent cleanup.
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), adminActionCleanupTimeout)
		defer cancel()
		_ = k.KubeExecStream(cleanupCtx, namespaceEtcds, podName, etcdAnalysisContainer,
			[]string{"rm", "-f", snapshotPath},
			io.Discard, io.Discard)
	}()

	// Step 1: Create the etcd snapshot.
	fmt.Fprintf(w, "Creating etcd snapshot on %s...\n", nodeName)

	var snapshotStderr bytes.Buffer
	snapshotErr := k.KubeExecStream(ctx, namespaceEtcds, podName, etcdAnalysisContainer,
		// Unset ETCDCTL_ENDPOINTS so etcdctl falls back to the loopback default
		// (127.0.0.1:2379). snapshot save requires exactly one endpoint, but the
		// container's env var lists all three members. etcd 3.5+ also treats having
		// both ETCDCTL_ENDPOINTS and --endpoints as a fatal conflict.
		// TLS is still negotiated via ETCDCTL_CACERT/CERT/KEY env vars.
		[]string{"sh", "-c", "unset ETCDCTL_ENDPOINTS; etcdctl snapshot save " + snapshotPath},
		newLimitedWriter(w, "snapshot stdout"),
		newLimitedWriter(&snapshotStderr, "snapshot stderr"),
	)

	// DATA-RACE GUARD: only read snapshotStderr when ctx.Err() == nil.
	// See KubeExecStream for the full data-race contract.
	if ctx.Err() == nil && snapshotStderr.Len() > 0 {
		fmt.Fprintf(w, "stderr:\n%s", snapshotStderr.String())
	}

	if snapshotErr != nil {
		fmt.Fprintf(w, "Snapshot failed: %v\n", snapshotErr)
		return
	}

	fmt.Fprintf(w, "Snapshot created. Starting analysis job...\n")

	// Step 2: Create a privileged ServiceAccount so the pod's privileged +
	// hostPath spec is admitted by OpenShift's SCC.
	cleanup, saErr, cleanupErr := createPrivilegedServiceAccount(ctx, log, saName, clusterName, saAccount, k)
	if saErr != nil {
		fmt.Fprintf(w, "Failed to create service account: %v\n", saErr)
		if cleanupErr != nil {
			fmt.Fprintf(w, "SA partial cleanup also failed: %v\n", cleanupErr)
		}
		return
	}
	defer func() {
		if err := cleanup(); err != nil {
			fmt.Fprintf(w, "SA cleanup failed: %v\n", err)
		}
	}()

	// Step 3: Build and run the analysis job.
	// runJobStream is called synchronously (not in a goroutine) so we block
	// here until the job completes or the context is cancelled.  A
	// nopWriteCloser prevents runJobStream's defer w.Close() from closing
	// the underlying writer prematurely.
	job := buildEtcdAnalysisJob(nodeName, filename, saName)
	runJobStream(ctx, k, job, nopWriteCloser{w})
}

// buildEtcdAnalysisJob builds the Job that runs octosql-etcd against the snapshot.
func buildEtcdAnalysisJob(nodeName, filename, saName string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-analysis-" + utilrand.String(5),
			Namespace: namespaceEtcds,
			Labels:    map[string]string{"app": "etcd-snapshot-analysis"},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointerutils.ToPtr(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: saName,
					NodeSelector:       map[string]string{"kubernetes.io/hostname": nodeName},
					Tolerations: []corev1.Toleration{
						{Effect: corev1.TaintEffectNoSchedule, Operator: corev1.TolerationOpExists},
						{Effect: corev1.TaintEffectNoExecute, Operator: corev1.TolerationOpExists},
					},
					Containers: []corev1.Container{
						{
							Name:    "analyzer",
							Image:   etcdAnalysisImage,
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								fmt.Sprintf("/usr/local/bin/analyze-snapshot.sh --delete /snapshot/%s", filename),
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointerutils.ToPtr(true),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "etcd-data", MountPath: "/snapshot"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "etcd-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: etcdSnapshotDir,
								},
							},
						},
					},
				},
			},
		},
	}
}
