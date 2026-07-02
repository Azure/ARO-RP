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

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("Content-Disposition", `attachment; filename="`+filename+`.kubeconfig"`)
	adminReply(log, w, header, b, nil)
}

// SECURITY: authorization for this endpoint is enforced upstream by the ACIS
// Geneva Action manifest that fronts it. The RP handler does NOT independently
// verify that the caller is in an "elevated" SRE group -- it trusts the URL
// suffix (/new vs /newelevated) plus the ACIS gate. This diverges from the
// portal binary, which additionally checks Entra group membership at mint
// time (stringutils.GroupsIntersect(elevatedGroupIDs, groups)). A
// misconfigured ACIS manifest that exposes /newelevated to non-elevated SREs
// will succeed here without further checks. Follow-up: mirror the portal's
// in-process group check once the caller's group claims are surfaced through
// SystemData or an equivalent context field.
func (f *frontend) _adminOpenShiftClusterKubeconfigNew(ctx context.Context, log *logrus.Entry, r *http.Request, elevated bool) ([]byte, error) {
	if f.portalServingCert == nil {
		return nil, api.NewCloudError(http.StatusServiceUnavailable, api.CloudErrorCodeInternalServerError, "", "Portal serving certificate is not available; the kubeconfig endpoint is disabled.")
	}

	// middleware.Validate (in the admin route chain) has already enforced a
	// lower-case URL.Path and validated every segment (subscription id, RG,
	// provider namespace, resource type, resource name), so we follow the
	// repo-standard idiom used by every other admin handler: derive the
	// cluster ARM ID by trimming the "/admin" prefix off r.URL.Path. The
	// other admin handlers already have the resource ID as their full path
	// after that trim; this handler additionally strips the trailing
	// "/kubeconfig/{new,newelevated}" action segment because we embed
	// resourceID into the kubeconfig server URL.
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	resourceID = resourceID[:strings.LastIndex(resourceID, "/kubeconfig/")]

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

	// Audit breadcrumb: this is the only structured record in AllServiceLogs
	// tying an issued admin-kubeconfig token back to the SRE that requested
	// it. Emit BEFORE the Cosmos write so a failed Create still leaves a
	// mint-attempt log line.
	log.WithFields(logrus.Fields{
		"username":    username,
		"resourceID":  resourceID,
		"elevated":    elevated,
		"portalDocID": token,
		"ttlSeconds":  portalDoc.TTL,
	}).Info("admin kubeconfig mint")

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
