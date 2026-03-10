package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (f *frontend) postAdminOpenShiftClusterEtcdKeyCount(w http.ResponseWriter, r *http.Request) {
	f.adminStreamAction(w, r, f._postAdminOpenShiftClusterEtcdKeyCount)
}

func (f *frontend) _postAdminOpenShiftClusterEtcdKeyCount(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	vmName := r.URL.Query().Get("vmName")
	if err := validateAdminVMName(vmName); err != nil {
		return err
	}

	k, resourceID, err := f.fetchClusterKubeActions(ctx, r, log)
	if err != nil {
		return err
	}

	log = log.WithFields(logrus.Fields{"resourceID": resourceID, "vmName": vmName})
	podName := "etcd-" + vmName
	opCtx, opCancel := context.WithTimeout(ctx, adminActionStreamTimeout)
	// execContainerStream installs its own panic recovery via utilrecover.Panic.
	go func() {
		defer opCancel()
		log.Info("admin etcdkeycount")
		// bash required: etcdKeyCountScript uses set -euo pipefail, which is bash-specific.
		execContainerStream(opCtx, log, k, namespaceEtcds, podName, etcdContainerName, []string{"bash", "-c"}, etcdKeyCountScript, writer)
	}()
	return nil
}
