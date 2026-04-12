package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
	"github.com/Azure/ARO-RP/test/util/bufferedpipe"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	bsTestResourceGroup   = "resourceGroupCluster"
	bsTestResourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/" + bsTestResourceGroup
	bsTestInfraID         = "infra"
	bsTestLBID            = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + bsTestResourceGroup + "/providers/Microsoft.Network/loadBalancers/" + bsTestInfraID + "-internal"
)

func newBSTestDoc(resourceGroupID, infraID string) *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key: "testkey",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				InfraID: infraID,
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "10.0.0.1",
				},
			},
		},
	}
}

func TestLogBootstrapNode(t *testing.T) {
	for _, tt := range []struct {
		name       string
		doc        *api.OpenShiftClusterDocument
		wantOutput []any
	}{
		{
			name:       "nil clients returns descriptive entry without panic",
			doc:        newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			wantOutput: []any{"lb or interface client missing"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			m := &manager{
				log: log,
				doc: tt.doc,
			}

			out, err := m.LogBootstrapNode(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantOutput != nil {
				for _, d := range deep.Equal(out, tt.wantOutput) {
					t.Error(d)
				}
			}
		})
	}
}

func TestSetupBootstrapNodeSSH(t *testing.T) {
	for _, tt := range []struct {
		name            string
		doc             *api.OpenShiftClusterDocument
		mockLB          func(*mock_armnetwork.MockLoadBalancersClient)
		mockNIC         func(*mock_armnetwork.MockInterfacesClient)
		wantErrContains string
	}{
		{
			name:            "empty InfraID returns error",
			doc:             newBSTestDoc(bsTestResourceGroupID, ""),
			wantErrContains: "infraID is not set",
		},
		{
			name: "empty APIServerPrivateEndpointIP returns error",
			doc: func() *api.OpenShiftClusterDocument {
				d := newBSTestDoc(bsTestResourceGroupID, bsTestInfraID)
				d.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP = ""
				return d
			}(),
			wantErrContains: "APIServerPrivateEndpointIP is not set",
		},
		{
			name: "LB Get failure returns error",
			doc:  newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{}, errors.New("lb get failed"))
			},
			mockNIC:         func(*mock_armnetwork.MockInterfacesClient) {},
			wantErrContains: "lb get failed",
		},
		{
			name: "NIC Get failure returns error",
			doc:  newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				lb := makeTestLBWithBootstrapConfig()
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: lb}, nil)
			},
			mockNIC: func(m *mock_armnetwork.MockInterfacesClient) {
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-bootstrap-nic", nil).
					Return(armnetwork.InterfacesClientGetResponse{}, errors.New("nic get failed"))
			},
			wantErrContains: "nic get failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := &manager{
				log: log,
				doc: tt.doc,
			}

			if tt.mockLB != nil {
				lbClient := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
				tt.mockLB(lbClient)
				m.loadBalancers = lbClient
			}
			if tt.mockNIC != nil {
				nicClient := mock_armnetwork.NewMockInterfacesClient(ctrl)
				tt.mockNIC(nicClient)
				m.armInterfaces = nicClient
			}

			_, err := m.setupBootstrapNodeSSH(ctx)

			if tt.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("want error containing %q, got %v", tt.wantErrContains, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWaitForBootstrapNodeSSH(t *testing.T) {
	const testAddress = "10.0.0.1:22"
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name            string
		setupMock       func(*mock_env.MockInterface)
		ctx             func() context.Context
		wantConn        bool
		wantErrContains string
	}{
		{
			name: "returns connection on first successful dial",
			setupMock: func(mockEnv *mock_env.MockInterface) {
				mockEnv.EXPECT().Now().Return(t0)
				c, _ := net.Pipe()
				mockEnv.EXPECT().DialContext(gomock.Any(), "tcp", testAddress).Return(c, nil)
			},
			ctx:      context.Background,
			wantConn: true,
		},
		{
			name: "returns error when total deadline is exceeded",
			setupMock: func(mockEnv *mock_env.MockInterface) {
				mockEnv.EXPECT().Now().Return(t0)
				mockEnv.EXPECT().DialContext(gomock.Any(), "tcp", testAddress).Return(nil, errors.New("connection refused"))
				mockEnv.EXPECT().Now().Return(t0.Add(4 * time.Minute))
			},
			ctx:             context.Background,
			wantErrContains: "did not become available within",
		},
		{
			name: "returns context error when context is already cancelled",
			setupMock: func(mockEnv *mock_env.MockInterface) {
				mockEnv.EXPECT().Now().Return(t0)
				mockEnv.EXPECT().DialContext(gomock.Any(), "tcp", testAddress).Return(nil, errors.New("connection refused"))
				mockEnv.EXPECT().Now().Return(t0.Add(1 * time.Minute))
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErrContains: context.Canceled.Error(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, log := testlog.New()
			mockEnv := mock_env.NewMockInterface(ctrl)
			tt.setupMock(mockEnv)

			m := &manager{log: log, env: mockEnv}

			conn, err := m.waitForBootstrapNodeSSH(tt.ctx(), testAddress)

			if tt.wantConn {
				if conn == nil {
					t.Error("expected a non-nil connection, got nil")
				} else {
					conn.Close()
				}
			}
			if tt.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("want error containing %q, got %v", tt.wantErrContains, err)
				}
			} else if !tt.wantConn && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnsureBootstrapLBConfig(t *testing.T) {
	for _, tt := range []struct {
		name            string
		lb              armnetwork.LoadBalancer
		wantChanged     bool
		wantErrContains string
		wantProbes      []string
		wantPools       []string
		wantRules       []string
	}{
		{
			name:        "empty LB gets all three resources added",
			lb:          makeMinimalLB(),
			wantChanged: true,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name:        "LB already fully configured is not modified",
			lb:          makeTestLBWithBootstrapConfig(),
			wantChanged: false,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name: "LB missing only the rule gets rule added",
			lb: func() armnetwork.LoadBalancer {
				lb := makeMinimalLB()
				lb.Properties.Probes = []*armnetwork.Probe{{Name: pointerutils.ToPtr(bootstrapNodeSSHProbeName)}}
				lb.Properties.BackendAddressPools = []*armnetwork.BackendAddressPool{{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)}}
				return lb
			}(),
			wantChanged: true,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name:            "nil properties returns error",
			lb:              armnetwork.LoadBalancer{},
			wantErrContains: "nil properties",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.New()
			m := &manager{log: log}

			got, err := m.ensureBootstrapNodeLBConfig(&tt.lb)

			if tt.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("want error containing %q, got %v", tt.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantChanged {
				t.Errorf("changed = %v, want %v", got, tt.wantChanged)
			}

			probeNames := namesFrom(lbProbes(tt.lb))
			for _, want := range tt.wantProbes {
				if !contains(probeNames, want) {
					t.Errorf("probe %q not found in %v", want, probeNames)
				}
			}
			poolNames := namesFrom(lbPools(tt.lb))
			for _, want := range tt.wantPools {
				if !contains(poolNames, want) {
					t.Errorf("pool %q not found in %v", want, poolNames)
				}
			}
			ruleNames := namesFrom(lbRules(tt.lb))
			for _, want := range tt.wantRules {
				if !contains(ruleNames, want) {
					t.Errorf("rule %q not found in %v", want, ruleNames)
				}
			}
		})
	}
}

func TestEnsureNICInBootstrapPool(t *testing.T) {
	poolID := bsTestLBID + "/backendAddressPools/" + bootstrapNodeBackendPool

	for _, tt := range []struct {
		name        string
		nic         armnetwork.Interface
		wantChanged bool
	}{
		{
			name:        "NIC with no pools gets pool added",
			nic:         makeTestNIC(nil),
			wantChanged: true,
		},
		{
			name:        "NIC already in pool is not modified",
			nic:         makeTestNIC([]string{poolID}),
			wantChanged: false,
		},
		{
			name:        "NIC in different pool gets bootstrap pool added",
			nic:         makeTestNIC([]string{bsTestLBID + "/backendAddressPools/other-pool"}),
			wantChanged: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureNICInBootstrapNodePool(&tt.nic, poolID)
			if got != tt.wantChanged {
				t.Errorf("changed = %v, want %v", got, tt.wantChanged)
			}
			if tt.wantChanged {
				found := false
				for _, ipc := range tt.nic.Properties.IPConfigurations {
					for _, p := range ipc.Properties.LoadBalancerBackendAddressPools {
						if p.ID != nil && strings.EqualFold(*p.ID, poolID) {
							found = true
						}
					}
				}
				if !found {
					t.Errorf("pool %q not found in NIC after update", poolID)
				}
			}
		})
	}
}

func TestBashQuote(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string",
			input: "hello world",
			want:  "'hello world'",
		},
		{
			name:  "empty string",
			input: "",
			want:  "''",
		},
		{
			name:  "embedded single quote",
			input: "it's",
			want:  "'it'\"'\"'s'",
		},
		{
			name:  "multiple embedded single quotes",
			input: "it's a 'test'",
			want:  "'it'\"'\"'s a '\"'\"'test'\"'\"''",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := bashQuote(tt.input)
			if got != tt.want {
				t.Errorf("bashQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTOFUHostKeyCallback(t *testing.T) {
	newSSHKey := func(t *testing.T) cryptossh.PublicKey {
		t.Helper()
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generating ed25519 key: %v", err)
		}
		pub, err := cryptossh.NewPublicKey(priv.Public())
		if err != nil {
			t.Fatalf("converting to ssh public key: %v", err)
		}
		return pub
	}

	key1 := newSSHKey(t)
	key2 := newSSHKey(t)

	for _, tt := range []struct {
		name     string
		calls    []cryptossh.PublicKey // keys presented in sequence
		wantErrs []bool                // whether each call should return an error
	}{
		{
			name:     "first key is accepted and recorded",
			calls:    []cryptossh.PublicKey{key1},
			wantErrs: []bool{false},
		},
		{
			name:     "same key accepted on second call",
			calls:    []cryptossh.PublicKey{key1, key1},
			wantErrs: []bool{false, false},
		},
		{
			name:     "different key rejected on second call",
			calls:    []cryptossh.PublicKey{key1, key2},
			wantErrs: []bool{false, true},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{}
			cb := m.toFUHostKeyCallback()
			for i, key := range tt.calls {
				err := cb("", nil, key)
				if (err != nil) != tt.wantErrs[i] {
					t.Errorf("call %d: got err=%v, wantErr=%v", i, err, tt.wantErrs[i])
				}
			}
		})
	}
}

// ---- helpers ----

func makeMinimalLB() armnetwork.LoadBalancer {
	return armnetwork.LoadBalancer{
		ID: pointerutils.ToPtr(bsTestLBID),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{ID: pointerutils.ToPtr(bsTestLBID + "/frontendIPConfigurations/public-lb-ip-v4")},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{},
			LoadBalancingRules:  []*armnetwork.LoadBalancingRule{},
			Probes:              []*armnetwork.Probe{},
		},
	}
}

func makeTestLBWithBootstrapConfig() armnetwork.LoadBalancer {
	lb := makeMinimalLB()
	lb.Properties.Probes = []*armnetwork.Probe{
		{Name: pointerutils.ToPtr(bootstrapNodeSSHProbeName)},
	}
	lb.Properties.BackendAddressPools = []*armnetwork.BackendAddressPool{
		{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)},
	}
	lb.Properties.LoadBalancingRules = []*armnetwork.LoadBalancingRule{
		{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)},
	}
	return lb
}

func makeTestNIC(poolIDs []string) armnetwork.Interface {
	pools := make([]*armnetwork.BackendAddressPool, 0, len(poolIDs))
	for _, id := range poolIDs {
		pools = append(pools, &armnetwork.BackendAddressPool{ID: &id})
	}
	return armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: pools,
					},
				},
			},
		},
	}
}

// Helpers to extract names from LB resource slices.

func lbProbes(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.Probes))
	for _, p := range lb.Properties.Probes {
		out = append(out, p.Name)
	}
	return out
}

func lbPools(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.BackendAddressPools))
	for _, p := range lb.Properties.BackendAddressPools {
		out = append(out, p.Name)
	}
	return out
}

func lbRules(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.LoadBalancingRules))
	for _, r := range lb.Properties.LoadBalancingRules {
		out = append(out, r.Name)
	}
	return out
}

func namesFrom(ptrs []*string) []string {
	out := make([]string, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// newTestSSHPair creates an in-process SSH client+server pair connected via
// bufferedpipe. The server accepts any client without authentication and dispatches
// session channels to handler. The returned client is ready to use; both sides
// are closed via t.Cleanup.
func newTestSSHPair(t *testing.T, handler func(cryptossh.Channel, <-chan *cryptossh.Request)) *cryptossh.Client {
	t.Helper()

	c1, c2 := bufferedpipe.New()
	t.Cleanup(func() { c1.Close(); c2.Close() })

	serverConfig := &cryptossh.ServerConfig{NoClientAuth: true}
	serverConfig.Config = cryptossh.Config{
		Ciphers:      utilssh.Ciphers(),
		KeyExchanges: utilssh.KexAlgorithms(),
		MACs:         utilssh.MACs(),
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating server host key: %v", err)
	}
	signer, err := cryptossh.NewSignerFromSigner(priv)
	if err != nil {
		t.Fatalf("creating server signer: %v", err)
	}
	serverConfig.AddHostKey(signer)

	go func() {
		_, channels, requests, err := cryptossh.NewServerConn(c2, serverConfig)
		if err != nil {
			return
		}
		go cryptossh.DiscardRequests(requests)
		for ch := range channels {
			if ch.ChannelType() != "session" {
				_ = ch.Reject(cryptossh.UnknownChannelType, "unknown channel type")
				continue
			}
			accepted, chanReqs, err := ch.Accept()
			if err != nil {
				return
			}
			go handler(accepted, chanReqs)
		}
	}()

	serverPublicKey, err := cryptossh.NewPublicKey(priv.Public())
	if err != nil {
		t.Fatalf("getting server public key: %v", err)
	}
	clientConfig := &cryptossh.ClientConfig{
		User:            "core",
		Auth:            []cryptossh.AuthMethod{cryptossh.Password("")},
		HostKeyCallback: cryptossh.FixedHostKey(serverPublicKey),
		Config: cryptossh.Config{
			Ciphers:      utilssh.Ciphers(),
			KeyExchanges: utilssh.KexAlgorithms(),
			MACs:         utilssh.MACs(),
		},
	}

	sshConn, chans, reqs, err := cryptossh.NewClientConn(c1, "", clientConfig)
	if err != nil {
		t.Fatalf("SSH client conn: %v", err)
	}
	t.Cleanup(func() { sshConn.Close() })

	return cryptossh.NewClient(sshConn, chans, reqs)
}

// execHandler returns a session handler that responds to the first exec request
// with the given stdout, stderr, and exit code, then closes the channel.
func execHandler(stdout, stderr []byte, exitCode uint32) func(cryptossh.Channel, <-chan *cryptossh.Request) {
	return func(ch cryptossh.Channel, reqs <-chan *cryptossh.Request) {
		defer ch.Close()
		for req := range reqs {
			if req.Type != "exec" {
				_ = req.Reply(false, nil)
				continue
			}
			_ = req.Reply(true, nil)
			_, _ = ch.Write(stdout)
			_, _ = ch.Stderr().Write(stderr)
			_ = ch.CloseWrite()
			exitPayload := make([]byte, 4)
			binary.BigEndian.PutUint32(exitPayload, exitCode)
			_, _ = ch.SendRequest("exit-status", false, exitPayload)
			return
		}
	}
}

func TestRunSSHCommand(t *testing.T) {
	for _, tt := range []struct {
		name           string
		handler        func(cryptossh.Channel, <-chan *cryptossh.Request)
		commandTimeout time.Duration
		checkEntries   func(t *testing.T, entries []logrus.Entry)
	}{
		{
			name:    "command with zero exit logs stdout",
			handler: execHandler([]byte("hello\n"), nil, 0),
			checkEntries: func(t *testing.T, entries []logrus.Entry) {
				t.Helper()
				for _, e := range entries {
					if e.Data["stream"] == "stdout" && e.Message == "hello" {
						return
					}
				}
				t.Errorf("expected a stdout log entry with message %q, got entries: %v", "hello", entries)
			},
		},
		{
			name:    "command with non-zero exit logs exit code warning",
			handler: execHandler(nil, nil, 1),
			checkEntries: func(t *testing.T, entries []logrus.Entry) {
				t.Helper()
				for _, e := range entries {
					if e.Level == logrus.WarnLevel {
						if code, ok := e.Data["exit_code"]; ok && code != 0 {
							return
						}
					}
				}
				t.Errorf("expected a warning log entry with non-zero exit_code, got entries: %v", entries)
			},
		},
		{
			name: "timed out command logs warning and returns nil",
			handler: func(ch cryptossh.Channel, reqs <-chan *cryptossh.Request) {
				defer ch.Close()
				for req := range reqs {
					if req.Type == "exec" {
						_ = req.Reply(true, nil)
						// Don't return; the for-range blocks until the client
						// closes the session (timer fires and calls sess.Close).
						continue
					}
					_ = req.Reply(false, nil)
				}
			},
			commandTimeout: 50 * time.Millisecond,
			checkEntries: func(t *testing.T, entries []logrus.Entry) {
				t.Helper()
				for _, e := range entries {
					if e.Level == logrus.WarnLevel && e.Message == "command timed out" {
						return
					}
				}
				t.Errorf("expected a %q warning log entry, got entries: %v", "command timed out", entries)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hook, log := testlog.New()

			client := newTestSSHPair(t, tt.handler)

			m := &manager{
				log:            log,
				commandTimeout: tt.commandTimeout,
			}

			err := m.runSSHCommand(client, "echo hello")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tt.checkEntries(t, hook.Entries)
		})
	}
}
