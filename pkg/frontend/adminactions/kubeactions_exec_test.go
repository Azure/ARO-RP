package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
)

// newExecTestServer starts a TLS httptest server speaking v4.channel.k8s.io.
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

	// Force HTTP/1.1 via ALPN; WebSocket requires HTTP/1.1.
	ts := httptest.NewUnstartedServer(wsServer)
	ts.TLS = &tls.Config{NextProtos: []string{"http/1.1"}}
	ts.StartTLS()
	t.Cleanup(ts.Close)

	// Encode the test server's self-signed cert as PEM for TLSClientConfig.CAData.
	cert := ts.Certificate()
	rc := &restclient.Config{
		Host: ts.URL,
		TLSClientConfig: restclient.TLSClientConfig{
			CAData: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}),
		},
	}
	return ts, rc
}

// execWebSocketURL returns a URL pointing at the test server's exec path.
func execWebSocketURL(ts *httptest.Server) *url.URL {
	u, err := url.Parse(ts.URL)
	if err != nil {
		panic(err)
	}
	u.Path = "/api/v1/namespaces/ns/pods/pod/exec"
	// Keep https; dialExecWebSocket rewrites to wss:// internally.
	u.Scheme = "https"
	return u
}

// sendFrame sends a single channel-prefixed WebSocket frame (1=stdout, 2=stderr, 3=exit-status).
func sendFrame(conn *websocket.Conn, channelID byte, data []byte) error {
	msg := make([]byte, 1+len(data))
	msg[0] = channelID
	copy(msg[1:], data)
	return websocket.Message.Send(conn, msg)
}

// runExecWithContext calls dialExecWebSocket+execWebSocketFrames, bypassing checkContainerReady.
func runExecWithContext(ctx context.Context, rc *restclient.Config, execURL *url.URL, stdout, stderr io.Writer) error {
	wsConn, tlsConn, err := dialExecWebSocket(ctx, rc, execURL)
	if err != nil {
		return err
	}
	var closeOnce sync.Once
	closeTLS := func() { closeOnce.Do(func() { tlsConn.Close() }) }
	defer closeTLS()
	return execWebSocketFrames(ctx, logrus.NewEntry(logrus.New()), wsConn, closeTLS, stdout, stderr)
}

// runExec is a convenience wrapper that uses a 5-second safety timeout.
func runExec(rc *restclient.Config, execURL *url.URL, stdout, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runExecWithContext(ctx, rc, execURL, stdout, stderr)
}

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
			name: "waiting container is reported as not ready with reason",
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
			wantErr: `container "etcdctl" in pod openshift-etcd/etcd-master-0 is not ready (waiting: ContainerCreating: pulling image)`,
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
			name: "terminated container is reported as not ready with reason",
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
									Reason:  "OOMKilled",
									Message: "memory limit exceeded",
								},
							},
						},
					},
				},
			},
			wantErr: `container "etcdctl" in pod openshift-etcd/etcd-master-0 is not ready (terminated: OOMKilled: memory limit exceeded)`,
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

// TestDialExecWebSocket_StdoutStderr verifies channel-1/2 frames route to correct writers.
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

// TestDialExecWebSocket_NonZeroExit verifies a Failure status frame surfaces as an error.
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

// TestDialExecWebSocket_NonZeroExit_EmptyMessage verifies that StatusFailure with an empty Message
// falls through to the hardcoded sentinel string.
func TestDialExecWebSocket_NonZeroExit_EmptyMessage(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		statusJSON, _ := json.Marshal(metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "",
		})
		_ = sendFrame(conn, 3, statusJSON)
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for StatusFailure with empty message; got nil")
	}
	if !strings.Contains(err.Error(), "exec command failed with no message from server") {
		t.Errorf("err = %q; want to contain 'exec command failed with no message from server'", err.Error())
	}
}

// TestDialExecWebSocket_ZeroLengthFrameSkipped verifies that a zero-length WebSocket frame is
// silently skipped and does not affect the final result.
func TestDialExecWebSocket_ZeroLengthFrameSkipped(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Send a zero-length frame; the client should skip it without error.
		_ = websocket.Message.Send(conn, []byte{})
		// Then send a normal success exit-status frame.
		statusJSON, _ := json.Marshal(metav1.Status{Status: metav1.StatusSuccess})
		_ = sendFrame(conn, 3, statusJSON)
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err != nil {
		t.Errorf("expected nil error after zero-length frame; got %v", err)
	}
}

// TestDialExecWebSocket_ContextCancellation verifies cancel unblocks even when the server hangs.
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

// TestDialExecWebSocket_PrematureEOF verifies close without status frame surfaces an error.
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

// TestDialExecWebSocket_UnexpectedChannelID verifies unknown channel IDs surface an error.
func TestDialExecWebSocket_UnexpectedChannelID(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Send a frame on channel 4, which is not part of the v4.channel.k8s.io protocol.
		_ = sendFrame(conn, 4, []byte("unexpected"))
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown channel ID; got nil")
	}
	if !strings.Contains(err.Error(), "unexpected exec channel ID") {
		t.Errorf("err = %q; want to contain 'unexpected exec channel ID'", err.Error())
	}
}

// errWriter is an io.Writer that always returns the given error on Write.
// A copy also exists in pkg/frontend/common_test.go; they cannot be shared because they are in different Go packages (frontend vs adminactions).
type errWriter struct{ err error }

func (e *errWriter) Write(_ []byte) (int, error) { return 0, e.err }

// TestDialExecWebSocket_WriterError verifies stdout write errors are propagated.
func TestDialExecWebSocket_WriterError(t *testing.T) {
	writeErr := errors.New("disk full")

	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// Send a stdout frame; the client's writer will fail.
		_ = sendFrame(conn, 1, []byte("some output"))
		// Wait for the connection to be torn down by the client.
		var msg []byte
		_ = websocket.Message.Receive(conn, &msg)
	})

	stdout := &errWriter{err: writeErr}
	var stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), stdout, &stderr)
	if err == nil {
		t.Fatal("expected write error; got nil")
	}
	if !errors.Is(err, writeErr) {
		t.Errorf("err = %v; want %v", err, writeErr)
	}
}

// TestDialExecWebSocket_MalformedStatusFrame verifies invalid JSON on channel 3 surfaces an error.
func TestDialExecWebSocket_MalformedStatusFrame(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		_ = sendFrame(conn, 3, []byte("not-valid-json"))
	})

	var stdout, stderr bytes.Buffer
	err := runExec(rc, execWebSocketURL(ts), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for malformed status frame; got nil")
	}
	if !strings.Contains(err.Error(), "malformed exit-status frame") {
		t.Errorf("err = %q; want to contain 'malformed exit-status frame'", err.Error())
	}
}

// TestDialExecWebSocket_HeartbeatTimeout verifies that an expired read deadline surfaces the
// "exec connection timed out" error from the heartbeat receive goroutine.
func TestDialExecWebSocket_HeartbeatTimeout(t *testing.T) {
	ts, rc := newExecTestServer(t, func(conn *websocket.Conn) {
		// Hold the connection without sending anything; client will time out.
		var msg []byte
		_ = websocket.Message.Receive(conn, &msg)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsConn, tlsConn, err := dialExecWebSocket(ctx, rc, execWebSocketURL(ts))
	if err != nil {
		t.Fatalf("dialExecWebSocket: %v", err)
	}
	// Pre-expire the read deadline so the first Receive returns a timeout error immediately.
	// heartbeatConn.Read only resets the deadline when n>0; a timeout read returns n=0,
	// so the pre-expired deadline fires without being extended by the wrapper.
	if err := tlsConn.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		tlsConn.Close()
		t.Fatalf("SetReadDeadline: %v", err)
	}
	var closeOnce sync.Once
	closeTLS := func() { closeOnce.Do(func() { tlsConn.Close() }) }
	defer closeTLS()

	var stdout, stderr bytes.Buffer
	err = execWebSocketFrames(ctx, logrus.NewEntry(logrus.New()), wsConn, closeTLS, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected timeout error; got nil")
	}
	if !strings.Contains(err.Error(), "exec connection timed out") {
		t.Errorf("err = %q; want to contain 'exec connection timed out'", err.Error())
	}
}

func TestPodLogFollowLimitSyncsWithExecOutputLimit(t *testing.T) {
	const execOutputLimit int64 = 1 << 20 // mirrors pkg/frontend/common.go
	if podLogFollowLimit != execOutputLimit+1 {
		t.Errorf("podLogFollowLimit = %d, want execOutputLimit+1 = %d", podLogFollowLimit, execOutputLimit+1)
	}
}
