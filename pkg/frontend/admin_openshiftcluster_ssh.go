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
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
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
	Master *int   `json:"master,omitempty"`
	VMName string `json:"vmName,omitempty"`
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
	if req.VMName != "" && req.Master != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "Specify either vmName or master, not both.")
	}
	if req.VMName == "" && req.Master == nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "Either vmName or master is required.")
	}
	if req.Master != nil && (*req.Master < 0 || *req.Master > 2) {
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

	var vmName string
	var port int
	if req.VMName != "" {
		vmName, port, err = f.resolveSSHVMTarget(ctx, log, doc, req.VMName)
		if err != nil {
			return nil, err
		}
	} else {
		// Transitional compatibility for the already-built Geneva Action.
		// New callers must send vmName so the RP can validate the exact VM.
		if err := f.checkSSHMasterPowered(ctx, log, doc, *req.Master); err != nil {
			return nil, err
		}
		port = 2200 + *req.Master
	}

	dbPortal, err := f.dbGroup.Portal()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	ssh := &api.SSH{}
	if vmName != "" {
		ssh.Master = port - 2200
		ssh.VMName = vmName
		ssh.Port = port
	} else {
		ssh.Master = *req.Master
	}

	password := dbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  password,
		TTL: int(adminSSHTTL / time.Second),
		Portal: &api.Portal{
			Username: username,
			ID:       resourceID,
			SSH:      ssh,
		},
	}

	// Audit trail. Emit BEFORE the Cosmos write so a failed Create still
	// leaves a creation-attempt record. Do NOT log the password: it is the
	// SSH bearer credential and would leak into Kusto.
	log.WithFields(logrus.Fields{
		"username":   username,
		"resourceID": resourceID,
		"vmName":     vmName,
		"port":       port,
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

func (f *frontend) resolveSSHVMTarget(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument, vmName string) (string, int, error) {
	if err := validateAdminVMName(vmName); err != nil {
		return "", 0, err
	}

	infraID := doc.OpenShiftCluster.Properties.InfraID
	isBootstrap := vmName == infraID+"-bootstrap"
	isMaster := strings.HasPrefix(vmName, infraID+"-master")
	if !isBootstrap && !isMaster {
		return "", 0, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmName", "VM must be a control-plane or bootstrap VM for this cluster.")
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return "", 0, err
	}
	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return "", 0, err
	}

	clusterRGName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vm, err := a.GetVirtualMachine(ctx, clusterRGName, vmName, mgmtcompute.InstanceView)
	if err != nil {
		if azureerrors.IsStatusNotFoundError(err) {
			return "", 0, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "vmName", fmt.Sprintf("Virtual machine %q was not found.", vmName))
		}
		return "", 0, err
	}
	powerState := masterPowerStateCode(vm)
	if powerState != "PowerState/running" {
		if powerState == "" {
			powerState = "unknown"
		}
		return "", 0, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmName", fmt.Sprintf("VM %q is not running (%s).", vmName, powerState))
	}
	if vm.NetworkProfile == nil || vm.NetworkProfile.NetworkInterfaces == nil || len(*vm.NetworkProfile.NetworkInterfaces) == 0 || (*vm.NetworkProfile.NetworkInterfaces)[0].ID == nil {
		return "", 0, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmName", fmt.Sprintf("VM %q has no network interface.", vmName))
	}

	selectedNICID := *(*vm.NetworkProfile.NetworkInterfaces)[0].ID
	nicName := stringutils.LastTokenByte(selectedNICID, '/')
	nicInfo, err := a.GetNetworkInterfaceSSHInfo(ctx, clusterRGName, nicName)
	if err != nil {
		if azureerrors.IsStatusNotFoundError(err) {
			return "", 0, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "vmName", fmt.Sprintf("Network interface %q for VM %q was not found.", nicName, vmName))
		}
		return "", 0, err
	}
	if !strings.EqualFold(nicInfo.ID, selectedNICID) {
		return "", 0, fmt.Errorf("network interface lookup for %q returned unexpected ID %q", nicName, nicInfo.ID)
	}
	if isMaster {
		masterSubnetID := doc.OpenShiftCluster.Properties.MasterProfile.SubnetID
		if len(nicInfo.SubnetIDs) == 0 {
			return "", 0, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmName", fmt.Sprintf("VM %q network interface has no subnet.", vmName))
		}
		for _, subnetID := range nicInfo.SubnetIDs {
			if !strings.EqualFold(subnetID, masterSubnetID) {
				return "", 0, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmName", fmt.Sprintf("VM %q is not attached to the cluster master subnet.", vmName))
			}
		}
	}

	lbName := infraID + "-internal"
	if doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		lbName = infraID + "-internal-lb"
	}
	lbID := strings.TrimSuffix(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, "/") + "/providers/Microsoft.Network/loadBalancers/" + lbName
	expectedPools := map[string]int{}
	if isBootstrap {
		expectedPools[strings.ToLower(lbID+"/backendAddressPools/bootstrap-ssh")] = 2199
	} else {
		for master, port := range []int{2200, 2201, 2202} {
			expectedPools[strings.ToLower(fmt.Sprintf("%s/backendAddressPools/ssh-%d", lbID, master))] = port
		}
	}

	matchingPoolCount := 0
	matchedPoolID := ""
	port := 0
	for _, poolID := range nicInfo.BackendPoolIDs {
		if matchedPort, ok := expectedPools[strings.ToLower(poolID)]; ok {
			matchingPoolCount++
			matchedPoolID = poolID
			port = matchedPort
		}
	}
	sshUnavailable := func(reason string) error {
		if isBootstrap {
			return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "vmName", fmt.Sprintf("Bootstrap SSH is unavailable for VM %q: %s. Run or re-run install-failure diagnostics to configure the bootstrap SSH route.", vmName, reason))
		}
		return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "vmName", fmt.Sprintf("Exact SSH routing is unavailable for VM %q: %s. Wait for control-plane SSH reconciliation to complete.", vmName, reason))
	}
	switch matchingPoolCount {
	case 0:
		return "", 0, sshUnavailable(fmt.Sprintf("its network interface is not attached to an SSH backend pool on load balancer %q", lbName))
	case 1:
	default:
		return "", 0, sshUnavailable(fmt.Sprintf("its network interface has %d SSH backend pool mappings on load balancer %q", matchingPoolCount, lbName))
	}

	routeStatus, err := a.GetSSHRouteStatus(
		ctx,
		clusterRGName,
		doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
		lbName,
		matchedPoolID,
		selectedNICID,
		doc.OpenShiftCluster.Properties.APIServerProfile.IntIP,
		int32(port),
	)
	if err != nil {
		if azureerrors.IsStatusNotFoundError(err) {
			return "", 0, sshUnavailable(fmt.Sprintf("load balancer %q or its SSH route was not found", lbName))
		}
		return "", 0, err
	}
	if routeStatus.LoadBalancingRuleCount != 1 {
		return "", 0, sshUnavailable(fmt.Sprintf("load balancer %q has %d matching SSH rules", lbName, routeStatus.LoadBalancingRuleCount))
	}
	if len(routeStatus.BackendPoolNetworkInterfaceIDs) != 1 || !strings.EqualFold(routeStatus.BackendPoolNetworkInterfaceIDs[0], selectedNICID) {
		return "", 0, sshUnavailable(fmt.Sprintf("backend pool has %d network interfaces and does not uniquely target the selected VM", len(routeStatus.BackendPoolNetworkInterfaceIDs)))
	}
	return vmName, port, nil
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
