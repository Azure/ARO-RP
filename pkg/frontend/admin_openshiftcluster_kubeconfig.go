package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

const (
	adminKubeconfigTTL = 6 * time.Hour
)

// postAdminOpenShiftClusterKubeconfigNew issues a non-elevated kubeconfig
// token. The resulting PortalDocument is consumed by the portal binary's
// /kubeconfig/proxy/* reverse proxy.
func (f *frontend) postAdminOpenShiftClusterKubeconfigNew(w http.ResponseWriter, r *http.Request) {
	f.adminOpenShiftClusterKubeconfigNew(w, r, false)
}

// postAdminOpenShiftClusterKubeconfigNewElevated issues an elevated kubeconfig
// token. The Geneva Action backing this route carries the JIT-Elevate policy.
func (f *frontend) postAdminOpenShiftClusterKubeconfigNewElevated(w http.ResponseWriter, r *http.Request) {
	f.adminOpenShiftClusterKubeconfigNew(w, r, true)
}

func (f *frontend) adminOpenShiftClusterKubeconfigNew(w http.ResponseWriter, r *http.Request, elevated bool) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	b, err := f._adminOpenShiftClusterKubeconfigNew(ctx, log, r, elevated)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	resourceName := chi.URLParam(r, "resourceName")
	filename := resourceName
	if elevated {
		filename += "-elevated"
	}

	// Write the body directly so the response is byte-identical to the
	// portal binary's /kubeconfig download (adminReply appends a trailing
	// newline).
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.kubeconfig"`)
	_, _ = w.Write(b)
}

// SECURITY: auth is enforced upstream by the ACIS Geneva Action manifest
// (URL suffix + manifest role). RP does NOT do an in-process group check.
// Uniform with other admin endpoints; portal binary diverges.
func (f *frontend) _adminOpenShiftClusterKubeconfigNew(ctx context.Context, log *logrus.Entry, r *http.Request, elevated bool) ([]byte, error) {
	if f.portalServingCert == nil {
		return nil, api.NewCloudError(http.StatusServiceUnavailable, api.CloudErrorCodeInternalServerError, "", "Portal serving certificate is not available; the kubeconfig endpoint is disabled.")
	}

	// Strip "/admin" prefix and the action suffix to leave the ARM ID for
	// the kubeconfig server URL. Suffix-trim (not LastIndex-slice) so a
	// router-table bug can't panic.
	actionSuffix := "/kubeconfig/new"
	if elevated {
		actionSuffix = "/kubeconfig/newelevated"
	}
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	if !strings.HasSuffix(resourceID, actionSuffix) {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unexpected admin kubeconfig URL path")
	}
	resourceID = strings.TrimSuffix(resourceID, actionSuffix)

	// Confirm the target cluster document exists before minting a
	// PortalDocument. Without this, a typo'd or already-deleted ARM ID
	// still gets a valid kubeconfig — the proxy would 404 later, but a
	// stray creation record and unused token would leak into Cosmos.
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	if _, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID)); err != nil {
		if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
				fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.",
					chi.URLParam(r, "resourceType"),
					chi.URLParam(r, "resourceName"),
					chi.URLParam(r, "resourceGroupName")))
		}
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	username := sreUsername(ctx)

	dbPortal, err := f.dbGroup.Portal()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	token := dbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  token,
		TTL: int(adminKubeconfigTTL / time.Second),
		Portal: &api.Portal{
			Username: username,
			ID:       resourceID,
			Kubeconfig: &api.Kubeconfig{
				Elevated: elevated,
			},
		},
	}

	// Audit trail. Emit BEFORE the Cosmos write so a failed Create still
	// leaves a creation-attempt record. Do NOT log the token: it is the
	// kubeconfig bearer credential and would leak into Kusto.
	log.WithFields(logrus.Fields{
		"username":   username,
		"resourceID": resourceID,
		"elevated":   elevated,
		"ttlSeconds": portalDoc.TTL,
	}).Info("admin kubeconfig create")

	if _, err := dbPortal.Create(ctx, portalDoc); err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	server := fmt.Sprintf("https://%s.admin.aro.azure.com%s/kubeconfig/proxy", strings.ToLower(f.env.Location()), resourceID)
	return makeAdminKubeconfig(server, token, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: f.portalServingCert.Raw,
	}))
}

// sreUsername returns the SRE UPN forwarded by ACIS in the ARM SystemData
// header.
func sreUsername(ctx context.Context) string {
	sd, ok := ctx.Value(middleware.ContextKeySystemData).(*api.SystemData)
	if !ok || sd == nil {
		return ""
	}
	if sd.LastModifiedBy != "" {
		return sd.LastModifiedBy
	}
	return sd.CreatedBy
}

func makeAdminKubeconfig(server, token string, caData []byte) ([]byte, error) {
	return json.MarshalIndent(&clientcmdv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "cluster",
				Cluster: clientcmdv1.Cluster{
					Server:                   server,
					CertificateAuthorityData: caData,
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: "user",
				AuthInfo: clientcmdv1.AuthInfo{
					Token: token,
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: "context",
				Context: clientcmdv1.Context{
					Cluster:   "cluster",
					Namespace: "default",
					AuthInfo:  "user",
				},
			},
		},
		CurrentContext: "context",
	}, "", "    ")
}
