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
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

type adminExecRequest struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
	Command   string `json:"command"`
}

func (f *frontend) postAdminOpenShiftClusterExec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	defer reader.Close()
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
		execContainerStream(opCtx, log, k, params.Namespace, params.PodName, params.Container, params.Command, writer)
	}()
	return nil
}

// execContainerStream streams a shell command's stdout/stderr to w; callable from other handlers.
func execContainerStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, namespace, podName, container, command string, w io.WriteCloser) {
	defer utilrecover.Panic(log)
	defer w.Close()

	fmt.Fprintf(w, "Executing in %s/%s/%s...\n", namespace, podName, container)

	var stderrBuf bytes.Buffer
	err := k.KubeExecStream(ctx, namespace, podName, container, []string{"sh", "-c", command},
		newLimitedWriter(w, "stdout"),
		newLimitedWriter(&stderrBuf, "stderr"),
	)

	// Skip stderr on cancellation: the output may be incomplete.
	if ctx.Err() == nil && stderrBuf.Len() > 0 {
		fmt.Fprintf(w, "stderr:\n%s", stderrBuf.String())
	}

	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintf(w, "Request cancelled.")
			return
		}
		log.WithFields(logrus.Fields{"namespace": namespace, "podName": podName, "container": container}).WithError(err).Warn("exec command failed")
		fmt.Fprintf(w, "Command failed: %v", err)
		return
	}
	fmt.Fprintf(w, "Done.")
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
