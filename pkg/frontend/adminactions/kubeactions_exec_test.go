package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
)

// newExecTestServer starts a TLS httptest server that speaks the
// v4.channel.k8s.io WebSocket subprotocol and returns both the server and a
// pre-configured rest.Config that trusts its self-signed certificate.
func newExecTestServer(t *testing.T, serverFn func(*websocket.Conn)) (*httptest.Server, *restclient.Config) {
	t.Helper()

	wsServer := websocket.Server{
		Config: websocket.Config{
			Protocol: []string{"v4.channel.k8s.io"},
		},
		// Accept any origin so our synthetic origin passes the handshake.
		Handshake: func(_ *websocket.Config, _ *http.Request) error {
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			serverFn(conn)
		},
	}

	// Force HTTP/1.1 via ALPN so Go 1.25's default h2 negotiation does not
	// prevent the WebSocket upgrade (WebSocket requires HTTP/1.1).
	ts := httptest.NewUnstartedServer(wsServer)
	ts.TLS = &tls.Config{NextProtos: []string{"http/1.1"}}
	ts.StartTLS()
	t.Cleanup(ts.Close)

	// Encode the test server's self-signed certificate as PEM so we can set
	// it in TLSClientConfig.CAData and have the client trust it.
	cert := ts.Certificate()
	var pemBuf bytes.Buffer
	pemBuf.WriteString("-----BEGIN CERTIFICATE-----\n")
	encoded := base64.StdEncoding.EncodeToString(cert.Raw)
	for len(encoded) > 0 {
		n := 64
		if n > len(encoded) {
			n = len(encoded)
		}
		pemBuf.WriteString(encoded[:n])
		pemBuf.WriteByte('\n')
		encoded = encoded[n:]
	}
	pemBuf.WriteString("-----END CERTIFICATE-----\n")

	rc := &restclient.Config{
		Host: ts.URL,
		TLSClientConfig: restclient.TLSClientConfig{
			CAData: pemBuf.Bytes(),
		},
	}
	return ts, rc
}

// execWebSocketURL returns a URL pointing at the test server's exec path with
// the https scheme (dialExecWebSocket switches it to wss:// internally).
func execWebSocketURL(ts *httptest.Server) *url.URL {
	u, err := url.Parse(ts.URL)
	if err != nil {
		panic(err)
	}
	u.Path = "/api/v1/namespaces/ns/pods/pod/exec"
	// Keep the https scheme - dialExecWebSocket rewrites to wss:// for the
	// WebSocket config, while using the host for the TCP dial.
	u.Scheme = "https"
	return u
}

// sendFrame sends a single channel-prefixed binary WebSocket frame.
// channelID: 1=stdout, 2=stderr, 3=exit-status.
func sendFrame(conn *websocket.Conn, channelID byte, data []byte) error {
	msg := make([]byte, 1+len(data))
	msg[0] = channelID
	copy(msg[1:], data)
	return websocket.Message.Send(conn, msg)
}

// driveReceiveLoop mirrors the goroutine inside KubeExecStream and drives the
// v4.channel.k8s.io receive loop to completion, returning the first error.
func driveReceiveLoop(conn *websocket.Conn, stdout, stderr io.Writer) error {
	for {
		var msg []byte
		if recvErr := websocket.Message.Receive(conn, &msg); recvErr != nil {
			if recvErr == io.EOF {
				return fmt.Errorf("connection closed before exit-status frame")
			}
			return recvErr
		}
		if len(msg) == 0 {
			continue
		}
		channelID, data := msg[0], msg[1:]
		switch channelID {
		case 1:
			if _, err := stdout.Write(data); err != nil {
				return err
			}
		case 2:
			if _, err := stderr.Write(data); err != nil {
				return err
			}
		case 3:
			var status metav1.Status
			if len(data) > 0 {
				if jsonErr := json.Unmarshal(data, &status); jsonErr != nil {
					return fmt.Errorf("malformed exit-status frame: %w", jsonErr)
				}
				if status.Status == metav1.StatusFailure {
					return errors.New(status.Message)
				}
			}
			return nil
		}
	}
}

// runExecWithContext calls dialExecWebSocket and then runs the same
// goroutine/select pattern as KubeExecStream, using the supplied context.
func runExecWithContext(ctx context.Context, rc *restclient.Config, execURL *url.URL, stdout, stderr io.Writer) error {
	wsConn, tlsConn, err := dialExecWebSocket(ctx, rc, execURL)
	if err != nil {
		return err
	}
	// Use sync.Once so that closing tlsConn is idempotent: the
	// context-cancellation path calls closeTLS() explicitly to unblock the
	// goroutine's Receive, and the deferred call ensures cleanup on all other
	// exit paths without double-closing.
	var closeOnce sync.Once
	closeTLS := func() { closeOnce.Do(func() { tlsConn.Close() }) }
	defer closeTLS()

	frameCh := make(chan execFrame, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(frameCh)
		defer wsConn.Close()
		for {
			var msg []byte
			if recvErr := websocket.Message.Receive(wsConn, &msg); recvErr != nil {
				if errors.Is(recvErr, io.EOF) {
					errCh <- fmt.Errorf("connection closed before exit-status frame")
				} else {
					errCh <- recvErr
				}
				return
			}
			if len(msg) == 0 {
				continue
			}
			channelID, data := msg[0], msg[1:]
			switch channelID {
			case 1, 2:
				select {
				case frameCh <- execFrame{channelID: channelID, data: data}:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			case 3:
				var status metav1.Status
				if len(data) > 0 {
					if jsonErr := json.Unmarshal(data, &status); jsonErr != nil {
						errCh <- fmt.Errorf("malformed exit-status frame: %w", jsonErr)
						return
					}
					if status.Status == metav1.StatusFailure {
						errCh <- errors.New(status.Message)
						return
					}
				}
				errCh <- nil
				return
			default:
				errCh <- fmt.Errorf("unexpected exec channel ID %d", channelID)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			closeTLS()
			return ctx.Err()
		case f, ok := <-frameCh:
			if !ok {
				return <-errCh
			}
			var writeErr error
			if f.channelID == 1 {
				_, writeErr = stdout.Write(f.data)
			} else {
				_, writeErr = stderr.Write(f.data)
			}
			if writeErr != nil {
				closeTLS()
				return writeErr
			}
		}
	}
}

// runExec is a convenience wrapper that uses a 5-second safety timeout.
func runExec(rc *restclient.Config, execURL *url.URL, stdout, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runExecWithContext(ctx, rc, execURL, stdout, stderr)
}

// ========================================================================
// Tests
// ========================================================================

func TestCheckContainerReady(t *testing.T) {
	const (
		ns        = "openshift-etcd"
		podName   = "etcd-master-0"
		container = "etcdctl"
	)

	for _, tt := range []struct {
		name      string
		pod       *corev1.Pod
		wantErr   string
	}{
		{
			name: "ready container returns nil",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: container, Ready: true},
					},
				},
			},
		},
		{
			name: "waiting container surfaces reason and message",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  container,
							Ready: false,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "ContainerCreating",
									Message: "pulling image",
								},
							},
						},
					},
				},
			},
			wantErr: `container "etcdctl" in pod openshift-etcd/etcd-master-0 is not ready: waiting (ContainerCreating: pulling image)`,
		},
		{
			name: "terminated container surfaces exit code and reason",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  container,
							Ready: false,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 137,
									Reason:   "OOMKilled",
								},
							},
						},
					},
				},
			},
			wantErr: `container "etcdctl" in pod openshift-etcd/etcd-master-0 is not ready: terminated (exit code 137: OOMKilled)`,
		},
		{
			name: "not-ready container with no state set",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: container, Ready: false},
					},
				},
			},
			wantErr: `container "etcdctl" in pod openshift-etcd/etcd-master-0 is not ready`,
		},
		{
			name: "container absent from status",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
				Status: corev1.PodStatus{
					Phase:             corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			wantErr: `container "etcdctl" not found in pod openshift-etcd/etcd-master-0 status (pod phase: Pending)`,
		},
		{
			name:    "pod not found",
			pod:     nil,
			wantErr: `getting pod openshift-etcd/etcd-master-0`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var kubecli *fake.Clientset
			if tt.pod != nil {
				kubecli = fake.NewSimpleClientset(tt.pod)
			} else {
				kubecli = fake.NewSimpleClientset()
			}
			k := &kubeActions{kubecli: kubecli}
			err := k.checkContainerReady(context.Background(), ns, podName, container)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error; got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %q; want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestDialExecWebSocket_StdoutStderr verifies that channel-1 and channel-2
// frames are routed to the correct writers.
func TestDialExecWebSocket_StdoutStderr(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		_ = sendFrame(conn, 1, []byte("hello stdout"))
		_ = sendFrame(conn, 2, []byte("hello stderr"))
		// Clean exit.
		statusJSON, _ := json.Marshal(metav1.Status{Status: metav1.StatusSuccess})
		_ = sendFrame(conn, 3, statusJSON)
	})

	var stdout, stderr bytes.Buffer
	if err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := stdout.String(); got != "hello stdout" {
		t.Errorf("stdout = %q; want %q", got, "hello stdout")
	}
	if got := stderr.String(); got != "hello stderr" {
		t.Errorf("stderr = %q; want %q", got, "hello stderr")
	}
}

// TestDialExecWebSocket_NonZeroExit verifies that a Failure status frame
// surfaces as an error containing the status message.
func TestDialExecWebSocket_NonZeroExit(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		statusJSON, _ := json.Marshal(metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "command terminated with exit code 1",
		})
		_ = sendFrame(conn, 3, statusJSON)
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-zero exit; got nil")
	}
	if !strings.Contains(err.Error(), "command terminated with exit code 1") {
		t.Errorf("err = %q; want to contain 'command terminated with exit code 1'", err.Error())
	}
}

// TestDialExecWebSocket_ContextCancellation verifies that cancelling the
// context unblocks the caller, even when the server hangs indefinitely.
func TestDialExecWebSocket_ContextCancellation(t *testing.T) {
	// serverReady is closed once the server-side goroutine is running.
	serverReady := make(chan struct{})

	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		close(serverReady)
		// Block until the connection is torn down from the client side.
		var msg []byte
		_ = websocket.Message.Receive(conn, &msg)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel the context as soon as the server signals it is ready.
	go func() {
		select {
		case <-serverReady:
			cancel()
		case <-time.After(5 * time.Second):
			// Safety valve - the test will time out via its own deadline.
		}
	}()

	var stdout, stderr bytes.Buffer
	err := runExecWithContext(ctx, rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected context.Canceled; got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
}

// TestDialExecWebSocket_PrematureEOF verifies that a server that closes the
// connection without sending a status frame surfaces an informative error.
func TestDialExecWebSocket_PrematureEOF(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		// Send some stdout, then abruptly close - no status frame.
		_ = sendFrame(conn, 1, []byte("partial"))
		conn.Close()
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for premature server close; got nil")
	}
	if !strings.Contains(err.Error(), "connection closed before exit-status frame") {
		t.Errorf("err = %q; want to contain 'connection closed before exit-status frame'", err.Error())
	}
	// Partial stdout must still have been delivered before the error.
	if got := stdout.String(); got != "partial" {
		t.Errorf("stdout = %q; want %q", got, "partial")
	}
}

// TestDialExecWebSocket_BearerTokenFile verifies that when rc.BearerTokenFile
// is set (and rc.BearerToken is empty), the token is read from disk and sent
// in the Authorization request header during the WebSocket upgrade.
func TestDialExecWebSocket_BearerTokenFile(t *testing.T) {
	const wantToken = "test-bearer-token-abc123"

	var gotAuth string
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		// Capture the Authorization header from the upgrade request.
		gotAuth = conn.Request().Header.Get("Authorization")
		// Send a clean exit so the client terminates.
		statusJSON, _ := json.Marshal(metav1.Status{Status: metav1.StatusSuccess})
		_ = sendFrame(conn, 3, statusJSON)
	})

	// Write the token to a temp file (with a trailing newline, as is
	// conventional for service-account token files).
	f, err := os.CreateTemp(t.TempDir(), "bearer-token-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(wantToken + "\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Ensure BearerToken is empty so that BearerTokenFile is used.
	rc.BearerToken = ""
	rc.BearerTokenFile = f.Name()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsConn, tlsConn, err := dialExecWebSocket(ctx, rc, execWebSocketURL(ts))
	if err != nil {
		t.Fatalf("dialExecWebSocket: %v", err)
	}
	// Drive the receive loop so the server handler finishes before we check.
	var stdout, stderr bytes.Buffer
	_ = driveReceiveLoop(wsConn, &stdout, &stderr)
	tlsConn.Close()
	wsConn.Close()

	want := "Bearer " + wantToken
	if gotAuth != want {
		t.Errorf("Authorization header = %q; want %q", gotAuth, want)
	}
}
