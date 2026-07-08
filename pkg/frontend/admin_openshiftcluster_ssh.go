package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
)

const adminSSHTTL = time.Minute

type adminSSHRequest struct {
	Master int `json:"master"`
}

type adminSSHResponse struct {
	Command  string `json:"command,omitempty"`
	Password string `json:"password,omitempty"`
}

// postAdminOpenShiftClusterSSHNewElevated mints a per-request SSH credential
// consumed by the portal binary's SSH reverse proxy. Elevation is JIT-gated
// upstream by the ACIS Geneva Action manifest that fronts this route.
func (f *frontend) postAdminOpenShiftClusterSSHNewElevated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resp, err := f._adminOpenShiftClusterSSHNewElevated(ctx, log, r)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	// Byte-for-byte parity with the portal binary's ssh.New response so
	// existing tooling can consume either endpoint interchangeably.
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	if err := enc.Encode(resp); err != nil {
		log.Warn(err)
	}
}

// SECURITY: auth is enforced upstream by the ACIS Geneva Action manifest
// (URL suffix + manifest role). RP does NOT do an in-process group check.
// Uniform with other admin endpoints; portal binary diverges.
func (f *frontend) _adminOpenShiftClusterSSHNewElevated(ctx context.Context, log *logrus.Entry, r *http.Request) (*adminSSHResponse, error) {
	if f.portalSSHHostPubKey == nil {
		return nil, api.NewCloudError(http.StatusServiceUnavailable, api.CloudErrorCodeInternalServerError, "", "Portal SSH host key is not available; the SSH endpoint is disabled.")
	}

	// Body middleware has already enforced Content-Type: application/json
	// and buffered the payload into ContextKeyBody.
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	var req adminSSHRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request body could not be parsed: %v.", err))
	}
	if req.Master < 0 || req.Master > 2 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.master", "master must be 0, 1, or 2.")
	}

	// Strip "/admin" prefix and the action suffix to leave the ARM ID. Suffix-
	// trim (not LastIndex-slice) so a router-table bug can't panic.
	const actionSuffix = "/ssh/newelevated"
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	if !strings.HasSuffix(resourceID, actionSuffix) {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unexpected admin SSH URL path")
	}
	resourceID = strings.TrimSuffix(resourceID, actionSuffix)

	username := sreUsername(ctx)

	dbPortal, err := f.dbGroup.Portal()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	password := dbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  password,
		TTL: int(adminSSHTTL / time.Second),
		Portal: &api.Portal{
			Username: username,
			ID:       resourceID,
			SSH: &api.SSH{
				Master: req.Master,
			},
		},
	}

	// Audit trail. Emit BEFORE the Cosmos write so a failed Create still
	// leaves a creation-attempt record. Do NOT log the password: it is the
	// SSH bearer credential and would leak into Kusto.
	log.WithFields(logrus.Fields{
		"username":   username,
		"resourceID": resourceID,
		"master":     req.Master,
		"ttlSeconds": portalDoc.TTL,
	}).Info("admin ssh create")

	if _, err := dbPortal.Create(ctx, portalDoc); err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	// Proxy's PasswordCallback splits Portal.Username on "@" and matches on
	// the LHS. Portal ssh.New sends the same LHS to the client here.
	sshUser := strings.SplitN(username, "@", 2)[0]
	hostname := fmt.Sprintf("%s.admin.aro.azure.com", strings.ToLower(f.env.Location()))
	command, err := adminCreateLoginCommand(sshUser, hostname, f.portalSSHHostPubKey)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return &adminSSHResponse{Command: command, Password: password}, nil
}

// adminSSHCommand mirrors pkg/portal/ssh.sshCommand but hard-codes the
// production SSH port (22). Duplicated intentionally: exporting the portal
// template would bind an ARM-facing handler to a portal-package internal.
const adminSSHCommand = "echo '{{ .KnownHostLine }}' > {{.Hostname}}_known_host ; " +
	"ssh " +
	"-o UserKnownHostsFile={{.Hostname}}_known_host " +
	"-o Ciphers={{ .Ciphers }} " +
	"-o HostKeyAlgorithms={{ .HostKeyAlgorithms }} " +
	"-o KexAlgorithms={{ .KexAlgorithms }} " +
	"-o MACs={{ .MACs }} {{.User}}@{{.Hostname}}"

func adminCreateLoginCommand(user, host string, publicKey cryptossh.PublicKey) (string, error) {
	line := knownhosts.Line([]string{host}, publicKey)
	tmp, err := template.New("command").Parse(adminSSHCommand)
	if err != nil {
		return "", err
	}
	type fields struct {
		User              string
		Hostname          string
		KnownHostLine     string
		Ciphers           string
		HostKeyAlgorithms string
		KexAlgorithms     string
		MACs              string
	}
	var buff bytes.Buffer
	err = tmp.Execute(&buff, fields{
		User:              user,
		Hostname:          host,
		KnownHostLine:     line,
		Ciphers:           utilssh.Ciphers()[0],
		HostKeyAlgorithms: utilssh.HostKeyAlgorithms()[0],
		KexAlgorithms:     utilssh.KexAlgorithms()[0],
		MACs:              utilssh.MACs()[0],
	})
	return buff.String(), err
}

// loadPortalSSHHostPubKey fetches the portal binary's SSH host key from the
// portal keyvault and derives its public key. Called once at frontend
// startup; on failure the admin SSH endpoint returns 503.
func loadPortalSSHHostPubKey(ctx context.Context, _env env.Interface) (cryptossh.PublicKey, error) {
	msiCredential, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	keyVaultPrefix := os.Getenv(encryption.KeyVaultPrefix)
	if keyVaultPrefix == "" {
		return nil, fmt.Errorf("%s env var not set", encryption.KeyVaultPrefix)
	}

	portalKeyvaultURI := azsecrets.URI(_env, env.PortalKeyvaultSuffix, keyVaultPrefix)
	secretsClient, err := azsecrets.NewClient(portalKeyvaultURI, msiCredential, _env.Environment().AzureClientOptions())
	if err != nil {
		return nil, fmt.Errorf("cannot create portal keyvault secrets client: %w", err)
	}

	serverSSHKey, err := secretsClient.GetSecret(ctx, env.PortalServerSSHKeySecretName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get portal server ssh key secret: %w", err)
	}

	b, err := azsecrets.ExtractBase64Value(serverSSHKey)
	if err != nil {
		return nil, err
	}

	// Portal binary stores an RSA host key in PKCS#1 form; mirror its parse.
	priv, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("cannot parse portal server ssh key: %w", err)
	}

	return cryptossh.NewPublicKey(&priv.PublicKey)
}
