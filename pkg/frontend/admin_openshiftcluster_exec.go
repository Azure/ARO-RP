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
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
)

type adminExecRequest struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
	Command   string `json:"command"`
}

// fn must close writer via a goroutine on success; must not close it on error.
func (f *frontend) adminStreamAction(w http.ResponseWriter, r *http.Request, fn func(context.Context, *http.Request, *logrus.Entry, io.WriteCloser) error) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = path.Dir(r.URL.Path) // strips action suffix so fetchClusterKubeActions can parse the resource path

	reader, writer := io.Pipe()
	defer reader.Close()
	err := fn(ctx, r, log, writer)
	if err != nil {
		_ = writer.CloseWithError(err)
	}
	var header http.Header
	if err == nil {
		header = http.Header{"Content-Type": []string{"text/plain"}}
	}
	f.streamResponder.AdminReplyStream(log, w, header, reader, err)
}

// Requires r.URL.Path to have been stripped of its action suffix; chi URL params are unaffected.
func (f *frontend) fetchClusterKubeActions(ctx context.Context, r *http.Request, log *logrus.Entry) (adminactions.KubeActions, string, error) {
	resType := chi.URLParam(r, "resourceType")
	resName := chi.URLParam(r, "resourceName")
	resGroupName := chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, "", api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, "", api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return nil, "", fmt.Errorf("fetching cluster document for %s: %w", resourceID, err)
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, "", fmt.Errorf("creating kube actions: %w", err)
	}

	return k, resourceID, nil
}

func (f *frontend) postAdminOpenShiftClusterExec(w http.ResponseWriter, r *http.Request) {
	f.adminStreamAction(w, r, f._postAdminOpenShiftClusterExec)
}

func (f *frontend) _postAdminOpenShiftClusterExec(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	body := ctx.Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			"The request body must not be empty.")
	}
	var params adminExecRequest
	if err := json.Unmarshal(body, &params); err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "",
			"Failed to parse request body.")
	}

	if err := validateAdminExec(params.Namespace, params.PodName, params.Container, params.Command); err != nil {
		return err
	}

	k, resourceID, err := f.fetchClusterKubeActions(ctx, r, log)
	if err != nil {
		return err
	}

	log = log.WithField("resourceID", resourceID)
	opCtx, opCancel := context.WithTimeout(ctx, adminActionStreamTimeout)
	// panic recovery is in execContainerStream
	go func() {
		defer opCancel()
		log.Info("admin exec")
		// sh: user commands must not depend on bash extensions
		execContainerStream(opCtx, log, k, params.Namespace, params.PodName, params.Container, []string{"sh", "-c"}, params.Command, writer)
	}()
	return nil
}

// Takes ownership of w; closes w before returning.
func execContainerStream(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, namespace, podName, container string, shell []string, command string, w io.WriteCloser) {
	defer utilrecover.Panic(log)
	defer w.Close()

	log = log.WithFields(logrus.Fields{"namespace": namespace, "podName": podName, "container": container})
	// write errors ignored; pipe reader may close early on client disconnect
	fmt.Fprintf(w, "Executing in %s/%s/%s...\n", namespace, podName, container)

	var stderrBuf bytes.Buffer
	err := k.KubeExecStream(ctx, namespace, podName, container, append(shell, command),
		newLimitedWriter(w, "stdout", log),
		newLimitedWriter(&stderrBuf, "stderr", log),
	)

	// Skip stderr on cancellation: the output may be incomplete.
	if ctx.Err() == nil && stderrBuf.Len() > 0 {
		fmt.Fprintf(newLimitedWriter(w, "stderr", log), "stderr:\n%s", stderrBuf.String())
	}

	if err != nil {
		if ctx.Err() != nil {
			log.WithError(err).Warn("exec cancelled")
			return
		}
		log.WithError(err).Warn("exec command failed")
		fmt.Fprintf(w, "Command failed: %v\n", err)
		return
	}
	if ctx.Err() != nil {
		return
	}
	fmt.Fprintf(w, "Done.\n")
}
