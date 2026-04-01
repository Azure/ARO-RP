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
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	etcdAnalysisImage        = "quay.io/redhat_emp1/octosql-etcd:latest" // TODO: move this to the arosvc ACR.
	etcdDataDir              = "/var/lib/etcd"
	etcdAnalysisSAPrefix     = "etcd-analysis-privileged"
	etcdAnalysisSuffixLength = 8
)

func (f *frontend) postAdminOpenShiftClusterEtcdAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	defer reader.Close()
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

	vmName := r.URL.Query().Get("vmName")
	if vmName == "" || !rxKubernetesString.MatchString(vmName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided vmName '%s' is invalid.", vmName))
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

	log = log.WithFields(logrus.Fields{"resourceID": resourceID, "vmName": vmName})
	opCtx, opCancel := context.WithCancel(ctx)
	go func() {
		defer opCancel()
		etcdAnalysisStream(opCtx, log, k, vmName, doc.OpenShiftCluster.Name, writer)
	}()
	return nil
}

// etcdAnalysisStream orchestrates the full etcd space analysis:
//  1. Creates an etcd snapshot via exec into the etcdctl container.
//     A best-effort rm -f of the snapshot file is deferred on all exit paths,
//     including partial files from a failed save.
//  2. Creates a privileged ServiceAccount for the analysis Job (reusing the
//     createPrivilegedServiceAccount helper from fixEtcd).
//  3. Launches an analysis Job against the snapshot file.
//
// Technique sourced from:
// https://github.com/openshift/configuration-anomaly-detection/blob/main/pkg/investigations/etcddatabasequotalowspace/analysis.go
func etcdAnalysisStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, vmName, clusterName string, w io.WriteCloser) {
	defer utilrecover.Panic(log)
	defer w.Close()

	podName := "etcd-" + vmName
	filename := "etcd_analysis_" + utilrand.String(etcdAnalysisSuffixLength) + ".snapshot"
	snapshotPath := etcdDataDir + "/" + filename
	saName := etcdAnalysisSAPrefix + "-" + utilrand.String(etcdAnalysisSuffixLength)
	saAccount := "system:serviceaccount:" + namespaceEtcds + ":" + saName

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), adminActionCleanupTimeout)
		defer cancel()
		if err := k.KubeExecStream(cleanupCtx, namespaceEtcds, podName, etcdContainerName,
			[]string{"rm", "-f", snapshotPath},
			io.Discard, io.Discard); err != nil {
			log.WithField("vmName", vmName).WithError(err).Warn("etcd snapshot cleanup failed")
			fmt.Fprintf(w, "Warning: snapshot cleanup failed: %v", err)
		}
	}()

	// Step 1: Create the etcd snapshot.
	fmt.Fprintf(w, "Creating etcd snapshot on %s...\n", vmName)

	var snapshotStderr bytes.Buffer
	snapshotErr := k.KubeExecStream(ctx, namespaceEtcds, podName, etcdContainerName,
		// Unset ETCDCTL_ENDPOINTS so etcdctl falls back to the loopback default
		// (127.0.0.1:2379). snapshot save requires exactly one endpoint, but the
		// container's env var lists all three members. etcd 3.5+ also treats having
		// both ETCDCTL_ENDPOINTS and --endpoints as a fatal conflict.
		// TLS is still negotiated via ETCDCTL_CACERT/CERT/KEY env vars.
		// Pass snapshotPath as $1 rather than interpolating it into the shell
		// string, so that any special characters in the path do not affect
		// parsing of the command string.
		[]string{"sh", "-c", "unset ETCDCTL_ENDPOINTS; etcdctl snapshot save \"$1\"", "--", snapshotPath},
		newLimitedWriter(w, "snapshot stdout"),
		newLimitedWriter(&snapshotStderr, "snapshot stderr"),
	)

	// Skip stderr on cancellation: the output may be incomplete.
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
			log.WithField("vmName", vmName).WithError(err).Error("etcd analysis SA cleanup failed; manual removal of privileged RBAC/SCC objects may be required")
			fmt.Fprintf(w, "SA cleanup failed: %v\n", err)
		}
	}()

	// Step 3: Run analysis job synchronously; nopWriteCloser prevents premature close.
	job := buildEtcdAnalysisJob(vmName, filename, saName)
	runJobStream(ctx, log, k, job, nopWriteCloser{w}, defaultJobRetryDelay)
}

// buildEtcdAnalysisJob builds the Job that runs octosql-etcd against the snapshot.
func buildEtcdAnalysisJob(vmName, filename, saName string) *batchv1.Job {
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
					NodeSelector:       map[string]string{"kubernetes.io/hostname": vmName},
					Tolerations: []corev1.Toleration{
						{Effect: corev1.TaintEffectNoSchedule, Operator: corev1.TolerationOpExists},
						{Effect: corev1.TaintEffectNoExecute, Operator: corev1.TolerationOpExists},
					},
					Containers: []corev1.Container{
						{
							Name:    "analyzer",
							Image:   etcdAnalysisImage,
							Command: []string{"/usr/local/bin/analyze-snapshot.sh"},
							Args:    []string{"--delete", "/snapshot/" + filename},
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
									Path: etcdDataDir,
								},
							},
						},
					},
				},
			},
		},
	}
}
