package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	bootstrapNodeSSHPort      = 2199
	bootstrapNodeBackendPool  = "bootstrap-ssh"
	bootstrapNodeSSHProbeName = "ssh"
)

// LogBootstrapNode reconfigures the ILB, establishes a single SSH connection to
// the bootstrap node, and runs general node-state and MCS diagnostics in sequence.
func (m *manager) LogBootstrapNode(ctx context.Context) (interface{}, error) {
	if m.armInterfaces == nil || m.loadBalancers == nil {
		return []interface{}{"lb or interface client missing"}, nil
	}
	client, err := m.setupBootstrapNodeSSH(ctx)
	if err != nil {
		return []interface{}{fmt.Sprintf("bootstrap SSH error: %v", err)}, nil
	}
	defer client.Close()
	m.logBootstrapNodeState(client)
	m.logBootstrapMCS(client)
	return []interface{}{}, nil
}

// logBootstrapNodeState runs general diagnostic commands using an established SSH client.
func (m *manager) logBootstrapNodeState(client *cryptossh.Client) {
	for _, cmd := range []string{
		"systemctl is-system-running",
		"systemctl list-units --no-pager --type service",
		"sudo crictl ps --all", // The "core" user can't access the CRI-O socket
		"sudo podman ps --all",
		"sudo ss -tlnp", // The "core" user can't see process names
	} {
		if err := m.runSSHCommand(client, cmd); err != nil {
			m.log.WithField("cmd", cmd).WithError(err).Error("running SSH command")
		}
	}
}

// logBootstrapMCS runs MCS diagnostic commands using an established SSH client.
func (m *manager) logBootstrapMCS(client *cryptossh.Client) {
	for _, cmd := range []string{
		// MCS runs as a CRI-O static pod on the bootstrap node (not podman).
		// crictl logs requires a container ID, so resolve it by name first.
		"sudo crictl logs --tail 100 $(sudo crictl ps -a --name machine-config-server -q | head -1)",
		"curl -vs --insecure --connect-timeout 10 --max-time 30 --head https://localhost:22623/config/master",
	} {
		if err := m.runSSHCommand(client, cmd); err != nil {
			m.log.WithField("cmd", cmd).WithError(err).Error("running SSH command")
		}
	}
}

// setupBootstrapNodeSSH reconfigures the internal load balancer to expose the
// bootstrap node on port bootstrapNodeSSHPort and returns an established SSH client.
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
		return nil, fmt.Errorf("reconfiguring ILB for bootstrap SSH: %w", err)
	}

	key, err := x509.ParsePKCS1PrivateKey(m.doc.OpenShiftCluster.Properties.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	signer, err := cryptossh.NewSignerFromSigner(key)
	if err != nil {
		return nil, fmt.Errorf("creating SSH signer: %w", err)
	}

	address := fmt.Sprintf("%s:%d", privateEndpointIP, bootstrapNodeSSHPort)
	conn, err := m.waitForBootstrapNodeSSH(ctx, address)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("establishing SSH connection to bootstrap: %w", err)
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
	log.Info("waiting for bootstrap SSH to become available")

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
			return nil, fmt.Errorf("bootstrap SSH at %s did not become available within %s: %w", address, totalTimeout, err)
		}

		log.WithError(err).Debugf("bootstrap SSH not yet available, retrying in %s", wait)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
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
			return fmt.Errorf("bootstrap host key mismatch: expected %s %s, got %s %s",
				m.bootstrapNodeHostKey.Type(), cryptossh.FingerprintSHA256(m.bootstrapNodeHostKey),
				key.Type(), cryptossh.FingerprintSHA256(key))
		}
		return nil
	}
}

func (m *manager) runSSHCommand(client *cryptossh.Client, cmd string) error {
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session: %w", err)
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	// Surface channel/connection errors but not non-zero exit codes;
	// the exit code is logged below so the caller can continue with the next command.
	runErr := sess.Run(cmd)
	var exitErr *cryptossh.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return runErr
	}

	logEntry := m.log.WithField("cmd", cmd)
	stdoutLog := logEntry.WithField("stream", "stdout")
	stderrLog := logEntry.WithField("stream", "stderr")
	for _, line := range strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n") {
		if line != "" {
			stdoutLog.Info(line)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(stderr.String(), "\n"), "\n") {
		if line != "" {
			stderrLog.Info(line)
		}
	}
	if exitErr != nil {
		logEntry.WithField("exit_code", exitErr.ExitStatus()).Info("command exited non-zero")
	}

	return nil
}

// ensureBootstrapNodeSSHAccess idempotently reconfigures the internal load balancer
// to route port bootstrapNodeSSHPort to the bootstrap node, then adds the bootstrap
// NIC to the backend pool.
//
// The bootstrap-ssh pool, probe, and LB rule are intentionally left in place
// after this function returns. The ILB is only reachable via the RP's private
// link endpoint (it is not publicly accessible), so leaving the rule behind
// poses no additional exposure. SRE can use it to SSH into the bootstrap node
// for manual troubleshooting of a failed install.
func (m *manager) ensureBootstrapNodeSSHAccess(ctx context.Context, resourceGroup, infraID string) error {
	lbName, err := m.ilbName(infraID)
	if err != nil {
		return err
	}

	lbResp, err := m.loadBalancers.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return fmt.Errorf("getting ILB %s: %w", lbName, err)
	}
	lb := lbResp.LoadBalancer

	if m.ensureBootstrapNodeLBConfig(&lb) {
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

func (m *manager) ilbName(infraID string) (string, error) {
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		return infraID + "-internal-lb", nil
	case api.ArchitectureVersionV2:
		return infraID + "-internal", nil
	default:
		return "", fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}
}

// ensureBootstrapNodeLBConfig adds the bootstrap-ssh backend pool, the ssh health
// probe, and the port-bootstrapNodeSSHPort load-balancing rule to lb if any of
// them are absent. It returns true if lb was modified.
func (m *manager) ensureBootstrapNodeLBConfig(lb *armnetwork.LoadBalancer) (changed bool) {
	if lb.Properties == nil {
		return false
	}
	hasProbe := false
	for _, p := range lb.Properties.Probes {
		if p.Name != nil && strings.EqualFold(*p.Name, bootstrapNodeSSHProbeName) {
			hasProbe = true
			break
		}
	}
	if !hasProbe {
		lb.Properties.Probes = append(lb.Properties.Probes, &armnetwork.Probe{
			Name: pointerutils.ToPtr(bootstrapNodeSSHProbeName),
			Properties: &armnetwork.ProbePropertiesFormat{
				Protocol:          pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
				Port:              pointerutils.ToPtr(int32(22)),
				IntervalInSeconds: pointerutils.ToPtr(int32(5)),
				NumberOfProbes:    pointerutils.ToPtr(int32(2)),
			},
		})
		changed = true
	}

	hasPool := false
	for _, p := range lb.Properties.BackendAddressPools {
		if p.Name != nil && strings.EqualFold(*p.Name, bootstrapNodeBackendPool) {
			hasPool = true
			break
		}
	}
	if !hasPool {
		lb.Properties.BackendAddressPools = append(lb.Properties.BackendAddressPools, &armnetwork.BackendAddressPool{
			Name: pointerutils.ToPtr(bootstrapNodeBackendPool),
		})
		changed = true
	}

	hasRule := false
	for _, r := range lb.Properties.LoadBalancingRules {
		if r.Name != nil && strings.EqualFold(*r.Name, bootstrapNodeBackendPool) {
			hasRule = true
			break
		}
	}
	if !hasRule {
		if lb.ID == nil || len(lb.Properties.FrontendIPConfigurations) == 0 || lb.Properties.FrontendIPConfigurations[0].ID == nil {
			m.log.Warn("skipping bootstrap LB rule: ILB missing ID or frontend IP configuration")
			return changed
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
				FrontendPort:         pointerutils.ToPtr(int32(bootstrapNodeSSHPort)),
				BackendPort:          pointerutils.ToPtr(int32(22)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
		})
		changed = true
	}

	return changed
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
