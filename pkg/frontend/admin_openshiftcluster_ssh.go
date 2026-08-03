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
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const adminSSHTTL = time.Minute

// rxSSHUsername bounds the identity local-part embedded into the returned ssh
// command to a conservative, shell-safe character set. A leading dash is
// disallowed so the command's "<user>@<host>" argument can't be parsed by ssh
// as a CLI option.
var rxSSHUsername = regexp.MustCompile(`^[a-zA-Z0-9._%+][a-zA-Z0-9._%+-]*$`)

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
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "master", "master must be 0, 1, or 2.")
	}

	// Strip "/admin" prefix and the action suffix to leave the ARM ID. Suffix-
	// trim (not LastIndex-slice) so a router-table bug can't panic.
	const actionSuffix = "/ssh/newelevated"
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	if !strings.HasSuffix(resourceID, actionSuffix) {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unexpected admin SSH URL path")
	}
	resourceID = strings.TrimSuffix(resourceID, actionSuffix)

	// Confirm the target cluster exists before minting. Otherwise a typo'd or
	// deleted resource ID leaves a stray PortalDocument (and a portal-side
	// "authentication succeeded" record) for a token that can never connect.
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	doc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
	if err != nil {
		if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
				fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.",
					chi.URLParam(r, "resourceType"),
					chi.URLParam(r, "resourceName"),
					chi.URLParam(r, "resourceGroupName")))
		}
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	// The minted command and the portal proxy's PasswordCallback both key off
	// the caller identity; an empty username yields a token the proxy can never
	// match. Fail fast rather than persist a dead PortalDocument.
	username := sreUsername(ctx)
	if username == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The caller identity could not be determined from the request.")
	}

	// sshUser (the local-part of the identity) is embedded into the returned
	// shell command and matched by the portal proxy's PasswordCallback. Reject
	// whitespace, shell metacharacters, or an empty local-part (e.g. "@contoso")
	// before minting, so the command can't break or be abused for injection by
	// automation that runs it.
	sshUser := strings.SplitN(username, "@", 2)[0]
	if !rxSSHUsername.MatchString(sshUser) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The caller identity contains unsupported characters.")
	}

	// Best-effort: if the requested master is powered off in Azure, reject now
	// with a clear message rather than mint a token that dies on a dial timeout
	// at connect. A transient Azure lookup failure must not block SSH access, so
	// anything short of a definitive "not running" is allowed through.
	if err := f.checkSSHMasterPowered(ctx, log, doc, req.Master); err != nil {
		return nil, err
	}

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

	// sshUser was validated above; the proxy's PasswordCallback matches it
	// against the LHS of Portal.Username.
	hostname := fmt.Sprintf("%s.admin.aro.azure.com", strings.ToLower(f.env.Location()))
	command, err := adminCreateLoginCommand(sshUser, hostname, f.portalSSHHostPubKey)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return &adminSSHResponse{Command: command, Password: password}, nil
}

// checkSSHMasterPowered rejects the request when the target master VM is not
// running in Azure, so the SRE gets a clear error instead of a dial timeout at
// connect. It is best-effort: subscription/client/lookup failures are logged
// and allowed through so a transient Azure blip can't break SSH access.
func (f *frontend) checkSSHMasterPowered(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, master int) error {
	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		log.Warnf("admin ssh: skipping master power-state check, cannot load subscription: %v", err)
		return nil
	}
	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		log.Warnf("admin ssh: skipping master power-state check, cannot build azure client: %v", err)
		return nil
	}
	clusterRGName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vmName := fmt.Sprintf("%s-master-%d", doc.OpenShiftCluster.Properties.InfraID, master)
	vm, err := a.GetVirtualMachine(ctx, clusterRGName, vmName, mgmtcompute.InstanceView)
	if err != nil {
		log.Warnf("admin ssh: skipping master power-state check, cannot get VM %s: %v", vmName, err)
		return nil
	}
	if ps := masterPowerStateCode(vm); ps != "" && ps != "PowerState/running" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "master",
			fmt.Sprintf("master-%d is not running (%s); power it on or choose a running master.", master, ps))
	}
	return nil
}

// masterPowerStateCode returns the "PowerState/*" status code from a VM instance
// view, or "" if none is present.
func masterPowerStateCode(vm mgmtcompute.VirtualMachine) string {
	if vm.InstanceView == nil || vm.InstanceView.Statuses == nil {
		return ""
	}
	for _, status := range *vm.InstanceView.Statuses {
		if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/") {
			return *status.Code
		}
	}
	return ""
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
