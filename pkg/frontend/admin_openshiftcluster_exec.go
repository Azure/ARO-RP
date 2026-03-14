package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

// execOutputLimit is the per-stream output cap for limitedWriter (exec stdout/stderr, pod logs).
const execOutputLimit int64 = 1 << 20

type adminExecRequest struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
	Command   string `json:"command"`
}

// limitedWriter wraps an io.Writer and silently truncates output once n bytes
// have been written, emitting a single truncation notice at that point.
type limitedWriter struct {
	w        io.Writer
	n        int64
	label    string
	exceeded bool
}

func newLimitedWriter(w io.Writer, label string) *limitedWriter {
	return &limitedWriter{w: w, n: execOutputLimit, label: label}
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.exceeded {
		return len(p), nil
	}
	toWrite := p
	truncated := false
	if int64(len(p)) > lw.n {
		toWrite = p[:lw.n]
		truncated = true
	}
	n, err := lw.w.Write(toWrite)
	lw.n -= int64(n)
	if truncated && err == nil && n == len(toWrite) {
		lw.exceeded = true
		_, _ = fmt.Fprintf(lw.w, "\n[%s truncated at 1 MiB]\n", lw.label)
	}
	return len(p), err
}

func (f *frontend) postAdminOpenShiftClusterExec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	err := f._postAdminOpenShiftClusterExec(ctx, r, log, writer)
	if err != nil {
		_ = writer.CloseWithError(err)
	}
	var header http.Header
	if err == nil {
		header = http.Header{"Content-Type": []string{"text/plain"}}
	}
	f.streamResponder.AdminReplyStream(log, w, header, reader, err)
}

func (f *frontend) _postAdminOpenShiftClusterExec(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			"The request body must not be empty.")
	}
	var params adminExecRequest
	if err := json.Unmarshal(body, &params); err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			fmt.Sprintf("Failed to parse request body: %v", err))
	}

	if err := validateAdminExec(params.Namespace, params.PodName, params.Container, params.Command); err != nil {
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

	go execContainerStream(ctx, k, params.Namespace, params.PodName, params.Container, params.Command, writer)
	return nil
}

// execContainerStream runs a shell command in a pod container, streaming stdout
// and stderr to w with progress and status lines. It is called in a goroutine
// by the HTTP handler, and may also be called directly by the etcdkeycount
// endpoint which has already resolved the cluster and kubeActions.
func execContainerStream(ctx context.Context, k adminactions.KubeActions, namespace, podName, container, command string, w io.WriteCloser) {
	defer w.Close()

	fmt.Fprintf(w, "Executing in %s/%s/%s...\n", namespace, podName, container)

	var stderrBuf bytes.Buffer
	err := k.KubeExecStream(ctx, namespace, podName, container, []string{"sh", "-c", command},
		newLimitedWriter(w, "stdout"),
		newLimitedWriter(&stderrBuf, "stderr"),
	)

	// DATA-RACE GUARD: on context cancellation, KubeExecStream closes the
	// WebSocket connection and returns immediately; the receive goroutine may
	// still be writing to stderrBuf. Only read it when ctx.Err() == nil.
	if ctx.Err() == nil && stderrBuf.Len() > 0 {
		fmt.Fprintf(w, "stderr:\n%s", stderrBuf.String())
	}

	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintf(w, "Request cancelled.\n")
			return
		}
		fmt.Fprintf(w, "Command failed: %v\n", err)
		return
	}
	fmt.Fprintf(w, "Done.\n")
}

func validateAdminExec(namespace, podName, container, command string) error {
	if namespace == "" || !rxKubernetesString.MatchString(namespace) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided namespace '%s' is invalid.", namespace))
	}
	if !utilnamespace.IsOpenShiftNamespace(namespace) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "",
			fmt.Sprintf("Access to the provided namespace '%s' is forbidden.", namespace))
	}
	if podName == "" || !rxKubernetesString.MatchString(podName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided pod name '%s' is invalid.", podName))
	}
	if container == "" || !rxKubernetesString.MatchString(container) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided container name '%s' is invalid.", container))
	}
	if command == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			"The provided command must not be empty.")
	}
	return nil
}
