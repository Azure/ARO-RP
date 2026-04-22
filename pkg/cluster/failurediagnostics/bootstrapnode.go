package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	bootstrapNodeLBSSHPort    = 2199
	bootstrapNodeBackendPool  = "bootstrap-ssh"
	bootstrapNodeSSHProbeName = "bootstrap-ssh"
	sshPort                   = 22
)

var bootstrapNodeSSHProbeProperties = &armnetwork.ProbePropertiesFormat{
	Protocol:          pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
	Port:              pointerutils.ToPtr(int32(sshPort)),
	IntervalInSeconds: pointerutils.ToPtr(int32(5)),
	NumberOfProbes:    pointerutils.ToPtr(int32(2)),
}

// LogBootstrapNode reconfigures the ILB, establishes an SSH connection to the
// bootstrap node, and runs the embedded diagnostic script.
func (m *manager) LogBootstrapNode(ctx context.Context) (interface{}, error) {
	if m.armInterfaces == nil || m.loadBalancers == nil {
		return []interface{}{"lb or interface client missing"}, nil
	}
	client, err := m.setupBootstrapNodeSSH(ctx)
	if err != nil {
		return []interface{}{fmt.Sprintf("bootstrap node SSH error: %v", err)}, nil
	}
	defer client.Close()
	if err := m.runDiagCommands(client, bootstrapNodeDiagCommands); err != nil {
		m.log.WithError(err).Error("running bootstrap node diagnostic commands")
	}
	return []interface{}{}, nil
}

// setupBootstrapNodeSSH reconfigures the internal load balancer to expose the
// bootstrap node on port bootstrapNodeLBSSHPort and returns an established SSH client.
// The caller is responsible for closing the client.
func (m *manager) setupBootstrapNodeSSH(ctx context.Context) (*cryptossh.Client, error) {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		return nil, fmt.Errorf("infraID is not set")
	}

	privateEndpointIP := m.doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP
	if privateEndpointIP == "" {
		return nil, fmt.Errorf("APIServerPrivateEndpointIP is not set")
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	if err := m.ensureBootstrapNodeSSHAccess(ctx, resourceGroup, infraID); err != nil {
		return nil, fmt.Errorf("reconfiguring ILB for bootstrap node SSH: %w", err)
	}

	key, err := x509.ParsePKCS1PrivateKey(m.doc.OpenShiftCluster.Properties.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	signer, err := cryptossh.NewSignerFromSigner(key)
	if err != nil {
		return nil, fmt.Errorf("creating SSH signer: %w", err)
	}

	address := fmt.Sprintf("%s:%d", privateEndpointIP, bootstrapNodeLBSSHPort)
	conn, err := m.waitForBootstrapNodeSSH(ctx, address)
	if err != nil {
		return nil, err
	}

	// Set a deadline on the underlying connection so the SSH handshake cannot hang indefinitely.
	if err := conn.SetDeadline(m.env.Now().Add(30 * time.Second)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting SSH handshake deadline: %w", err)
	}
	sshConn, chans, reqs, err := cryptossh.NewClientConn(conn, address, &cryptossh.ClientConfig{
		User: "core",
		Auth: []cryptossh.AuthMethod{
			cryptossh.PublicKeys(signer),
		},
		// The bootstrap node generates its SSH host key at first boot (via
		// sshd_keygen). The key is not embedded in the ignition config and
		// is never transmitted to the RP, so we cannot know it in advance.
		// We implement TOFU: accept and record the key on the first
		// connection, then verify it on any subsequent connections within
		// the same diagnostic run.
		HostKeyCallback:   m.toFUHostKeyCallback(),
		HostKeyAlgorithms: utilssh.HostKeyAlgorithms(),
		Config: cryptossh.Config{
			KeyExchanges: utilssh.KexAlgorithms(),
			Ciphers:      utilssh.Ciphers(),
			MACs:         utilssh.MACs(),
		},
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("establishing SSH connection to bootstrap node: %w", err)
	}
	// Clear the handshake deadline so it doesn't affect subsequent sessions.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("clearing SSH handshake deadline: %w", err)
	}

	return cryptossh.NewClient(sshConn, chans, reqs), nil
}

// waitForBootstrapNodeSSH polls the bootstrap SSH address until the LB health probe
// has passed and traffic is routed, returning the open connection for reuse.
// The Azure LB will not route traffic until the TCP probe on port 22 has
// succeeded twice (2 × 5 s = 10 s minimum after NIC association). Dropped
// packets mean a failed dial may block for the full per-attempt timeout, so
// a short per-attempt deadline is used alongside exponential backoff.
func (m *manager) waitForBootstrapNodeSSH(ctx context.Context, address string) (net.Conn, error) {
	const (
		dialTimeout  = 10 * time.Second
		initialWait  = 5 * time.Second
		maxWait      = 30 * time.Second
		totalTimeout = 3 * time.Minute
	)

	log := m.log.WithField("address", address)
	log.Info("waiting for bootstrap node SSH to become available")

	deadline := m.env.Now().Add(totalTimeout)
	wait := initialWait

	for {
		dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
		conn, err := m.env.DialContext(dialCtx, "tcp", address)
		cancel()

		if err == nil {
			return conn, nil
		}

		if m.env.Now().After(deadline) {
			return nil, fmt.Errorf("bootstrap node SSH at %s did not become available within %s: %w", address, totalTimeout, err)
		}

		log.WithError(err).Debugf("bootstrap node SSH not yet available, retrying in %s", wait)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}

		wait *= 2
		if wait > maxWait {
			wait = maxWait
		}
	}
}

// toFUHostKeyCallback returns an SSH HostKeyCallback that implements
// Trust-On-First-Use: the first host key seen is recorded on the manager and
// accepted; all subsequent connections must present the same key.
func (m *manager) toFUHostKeyCallback() cryptossh.HostKeyCallback {
	return func(_ string, _ net.Addr, key cryptossh.PublicKey) error {
		if m.bootstrapNodeHostKey == nil {
			m.bootstrapNodeHostKey = key
			return nil
		}
		if !bytes.Equal(m.bootstrapNodeHostKey.Marshal(), key.Marshal()) {
			return fmt.Errorf("bootstrap node host key mismatch: expected %s %s, got %s %s",
				m.bootstrapNodeHostKey.Type(), cryptossh.FingerprintSHA256(m.bootstrapNodeHostKey),
				key.Type(), cryptossh.FingerprintSHA256(key))
		}
		return nil
	}
}

// runDiagCommands unmarshals a JSON array of command strings and runs each
// command in its own SSH session via bash -c, logging stdout, stderr, and
// exit code per command.
func (m *manager) runDiagCommands(client *cryptossh.Client, commandsJSON string) error {
	var commands []string
	if err := json.Unmarshal([]byte(commandsJSON), &commands); err != nil {
		return fmt.Errorf("parsing diagnostic commands: %w", err)
	}
	for _, cmd := range commands {
		if err := m.runSSHCommand(client, cmd); err != nil {
			return err
		}
	}
	return nil
}

const commandTimeout = 2 * time.Minute

// runSSHCommand runs a single command in a new SSH session and logs its
// stdout, stderr, and exit code. Non-zero exit codes are logged but do not
// cause an error return; only SSH channel/connection failures are returned.
// Each command is bounded by commandTimeout to prevent hangs.
func (m *manager) runSSHCommand(client *cryptossh.Client, cmd string) error {
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session: %w", err)
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	// Close the session after commandTimeout to prevent indefinite hangs.
	// timedOut is set before sess.Close() in the timer goroutine; the
	// session close causes sess.Run to return, establishing the
	// happens-before edge so timedOut can be read safely afterwards.
	timeout := m.commandTimeout
	if timeout == 0 {
		timeout = commandTimeout
	}
	var timedOut atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		timedOut.Store(true)
		sess.Close()
	})
	runErr := sess.Run("bash -c " + bashQuote(cmd))
	timer.Stop()
	var exitErr *cryptossh.ExitError
	if runErr != nil && !timedOut.Load() && !errors.As(runErr, &exitErr) {
		return runErr
	}

	logEntry := m.log.WithField("cmd", cmd)
	if timedOut.Load() {
		logEntry.Warn("command timed out")
	}
	for _, line := range strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n") {
		if line != "" {
			logEntry.WithField("stream", "stdout").Info(line)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(stderr.String(), "\n"), "\n") {
		if line != "" {
			logEntry.WithField("stream", "stderr").Info(line)
		}
	}
	if exitErr != nil {
		logEntry.WithField("exit_code", exitErr.ExitStatus()).Warn("command exited non-zero")
	}

	return nil
}

// bashQuote wraps s in single quotes for use as a bash -c argument,
// escaping any embedded single quotes.
func bashQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// ensureBootstrapNodeSSHAccess idempotently reconfigures the internal load balancer
// to route port bootstrapNodeLBSSHPort to the bootstrap node, then adds the bootstrap
// NIC to the backend pool.
//
// The bootstrap-ssh pool, probe, and LB rule are intentionally left in place
// after this function returns. The ILB is only reachable via the RP's private
// link endpoint (it is not publicly accessible), so leaving the rule behind
// poses no additional exposure. SRE can use it to SSH into the bootstrap node
// for manual troubleshooting of a failed install.
func (m *manager) ensureBootstrapNodeSSHAccess(ctx context.Context, resourceGroup, infraID string) error {
	lbName := infraID + "-internal"

	lbResp, err := m.loadBalancers.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return fmt.Errorf("getting ILB %s: %w", lbName, err)
	}
	lb := lbResp.LoadBalancer

	changed, err := m.ensureBootstrapNodeLBConfig(&lb)
	if err != nil {
		return fmt.Errorf("configuring ILB %s: %w", lbName, err)
	}
	if changed {
		if err := m.loadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb, nil); err != nil {
			return fmt.Errorf("updating ILB %s: %w", lbName, err)
		}
	}

	bootstrapNICName := infraID + "-bootstrap-nic"
	nicResp, err := m.armInterfaces.Get(ctx, resourceGroup, bootstrapNICName, nil)
	if err != nil {
		return fmt.Errorf("getting bootstrap NIC %s: %w", bootstrapNICName, err)
	}
	nic := nicResp.Interface

	if lb.ID == nil {
		return fmt.Errorf("ILB %s has no ID", lbName)
	}
	bootstrapPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, bootstrapNodeBackendPool)
	if ensureNICInBootstrapNodePool(&nic, bootstrapPoolID) {
		if err := m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, bootstrapNICName, nic, nil); err != nil {
			return fmt.Errorf("updating bootstrap NIC: %w", err)
		}
	}

	return nil
}

// ensureBootstrapNodeLBConfig adds the bootstrap-ssh backend pool, the ssh health
// probe, and the port-bootstrapNodeLBSSHPort load-balancing rule to lb if any of
// them are absent. It returns true if lb was modified.
func (m *manager) ensureBootstrapNodeLBConfig(lb *armnetwork.LoadBalancer) (changed bool, err error) {
	if lb.Properties == nil {
		return false, fmt.Errorf("ILB has nil properties")
	}

	if !slices.ContainsFunc(lb.Properties.Probes, func(p *armnetwork.Probe) bool {
		return p.Name != nil && strings.EqualFold(*p.Name, bootstrapNodeSSHProbeName)
	}) {
		lb.Properties.Probes = append(lb.Properties.Probes, &armnetwork.Probe{
			Name:       pointerutils.ToPtr(bootstrapNodeSSHProbeName),
			Properties: bootstrapNodeSSHProbeProperties,
		})
		changed = true
	}

	if !slices.ContainsFunc(lb.Properties.BackendAddressPools, func(p *armnetwork.BackendAddressPool) bool {
		return p.Name != nil && strings.EqualFold(*p.Name, bootstrapNodeBackendPool)
	}) {
		lb.Properties.BackendAddressPools = append(lb.Properties.BackendAddressPools, &armnetwork.BackendAddressPool{
			Name: pointerutils.ToPtr(bootstrapNodeBackendPool),
		})
		changed = true
	}

	if !slices.ContainsFunc(lb.Properties.LoadBalancingRules, func(r *armnetwork.LoadBalancingRule) bool {
		return r.Name != nil && strings.EqualFold(*r.Name, bootstrapNodeBackendPool)
	}) {
		if lb.ID == nil || len(lb.Properties.FrontendIPConfigurations) == 0 || lb.Properties.FrontendIPConfigurations[0].ID == nil {
			return changed, fmt.Errorf("ILB missing ID or frontend IP configuration")
		}
		lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules, &armnetwork.LoadBalancingRule{
			Name: pointerutils.ToPtr(bootstrapNodeBackendPool),
			Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork.SubResource{
					ID: lb.Properties.FrontendIPConfigurations[0].ID,
				},
				BackendAddressPool: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, bootstrapNodeBackendPool)),
				},
				Probe: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("%s/probes/%s", *lb.ID, bootstrapNodeSSHProbeName)),
				},
				Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(bootstrapNodeLBSSHPort)),
				BackendPort:          pointerutils.ToPtr(int32(sshPort)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
		})
		changed = true
	}

	return changed, nil
}

// ensureNICInBootstrapNodePool adds bootstrapPoolID to the NIC's backend address
// pools if it is not already present. Returns true if the NIC was modified.
func ensureNICInBootstrapNodePool(nic *armnetwork.Interface, bootstrapPoolID string) (changed bool) {
	if nic == nil || nic.Properties == nil {
		return false
	}
	for _, ipc := range nic.Properties.IPConfigurations {
		if ipc == nil || ipc.Properties == nil {
			continue
		}
		if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(pool *armnetwork.BackendAddressPool) bool {
			return pool.ID != nil && strings.EqualFold(*pool.ID, bootstrapPoolID)
		}) {
			ipc.Properties.LoadBalancerBackendAddressPools = append(
				ipc.Properties.LoadBalancerBackendAddressPools,
				&armnetwork.BackendAddressPool{ID: pointerutils.ToPtr(bootstrapPoolID)},
			)
			changed = true
		}
	}
	return changed
}
