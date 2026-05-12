package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilrecover "github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// KubeActions are those that involve k8s objects, and thus depend upon k8s clients being createable
type KubeActions interface {
	KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	KubeList(ctx context.Context, groupKind, namespace string, labelSelector ...string) ([]byte, error)
	KubeCreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	KubeDelete(ctx context.Context, groupKind, namespace, name string, force bool, propagationPolicy *metav1.DeletionPropagation) error
	KubeExecStream(ctx context.Context, namespace, pod, container string, command []string, stdout, stderr io.Writer) error
	KubeFollowPodLogs(ctx context.Context, namespace, podName, containerName string, w io.Writer) error
	ResolveGVR(groupKind string, optionalVersion string) (schema.GroupVersionResource, error)
	CordonNode(ctx context.Context, nodeName string, unschedulable bool) error
	DrainNode(ctx context.Context, nodeName string) error
	DrainNodeWithRetries(ctx context.Context, nodeName string) error
	ApproveCsr(ctx context.Context, csrName string) error
	ApproveAllCsrs(ctx context.Context) error
	KubeGetPodLogs(ctx context.Context, namespace, name, containerName string) ([]byte, error)
	KubeWatch(ctx context.Context, o *unstructured.Unstructured, label string) (watch.Interface, error)
	TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]PodMetrics, error)
	TopNodes(ctx context.Context, restConfig *restclient.Config) ([]NodeMetrics, error)
	CheckAPIServerReadyz(ctx context.Context) error
}

type kubeActions struct {
	log *logrus.Entry
	oc  *api.OpenShiftCluster

	mapper meta.RESTMapper

	dyn     dynamic.Interface
	kubecli kubernetes.Interface
	rc      *restclient.Config
}

// NewKubeActions returns a kubeActions
func NewKubeActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (KubeActions, error) {
	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}

	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, err
	}

	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	kubecli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &kubeActions{
		log: log,
		oc:  oc,

		mapper: mapper,

		dyn:     dyn,
		kubecli: kubecli,
		rc:      restConfig,
	}, nil
}

func (k *kubeActions) KubeGetPodLogs(ctx context.Context, namespace, podName, containerName string) ([]byte, error) {
	var limit int64 = 52428800
	opts := corev1.PodLogOptions{Container: containerName, LimitBytes: &limit}
	return k.kubecli.CoreV1().Pods(namespace).GetLogs(podName, &opts).Do(ctx).Raw()
}

const (
	execHeartbeatPeriod   = 5 * time.Second
	execHeartbeatDeadline = 61 * time.Second // 1s past the typical 60s API server idle timeout
	execPingWriteTimeout  = 10 * time.Second
)

// sends a WebSocket ping control frame
var pingCodec = websocket.Codec{
	Marshal: func(v interface{}) ([]byte, byte, error) {
		return nil, websocket.PingFrame, nil
	},
}

// resets the read deadline on every successful read so pong frames extend the heartbeat window.
type heartbeatConn struct {
	net.Conn
	deadline time.Duration
}

func (c *heartbeatConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		_ = c.SetReadDeadline(time.Now().Add(c.deadline))
	}
	return n, err
}

// records the first recorderBufSize bytes read to recover the HTTP error body on a failed WebSocket upgrade.
const recorderBufSize = 4096

type recorderConn struct {
	net.Conn
	buf [recorderBufSize]byte
	n   int
}

func (r *recorderConn) Read(b []byte) (int, error) {
	n, err := r.Conn.Read(b)
	if n > 0 && r.n < recorderBufSize {
		r.n += copy(r.buf[r.n:], b[:n])
	}
	return n, err
}

// a v4.channel.k8s.io data frame from the receive goroutine.
type execFrame struct {
	channelID byte
	data      []byte
}

// Wrap command in []string{"sh", "-c", cmd} to use shell features.
func (k *kubeActions) KubeExecStream(ctx context.Context, namespace, pod, container string, command []string, stdout, stderr io.Writer) error {
	req := k.kubecli.CoreV1().RESTClient().Get().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   command,
			Container: container,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, kubescheme.ParameterCodec)

	wsConn, tlsConn, err := dialExecWebSocket(ctx, k.rc, req.URL())
	if err != nil {
		return err
	}
	// idempotent: called on ctx cancellation and by defer
	var closeOnce sync.Once
	closeTLS := func() { closeOnce.Do(func() { tlsConn.Close() }) }
	defer closeTLS()

	return execWebSocketFrames(ctx, k.log, wsConn, closeTLS, stdout, stderr)
}

// closeTLS must be idempotent; it's called from multiple code paths.
func execWebSocketFrames(ctx context.Context, log *logrus.Entry, wsConn *websocket.Conn, closeTLS func(), stdout, stderr io.Writer) error {
	// v4.channel.k8s.io: ch1=stdout, ch2=stderr, ch3=exit status (metav1.Status JSON)
	frameCh := make(chan execFrame, 16)
	errCh := make(chan error, 1) // capacity 1; the goroutine sends exactly once before terminating

	// pingStop is closed when execWebSocketFrames returns, stopping the ping goroutine.
	pingStop := make(chan struct{})
	defer close(pingStop)

	// pings every execHeartbeatPeriod; heartbeatConn resets the deadline on pong receipt so no pong handler is needed
	go func() {
		defer utilrecover.Panic(log)
		t := time.NewTicker(execHeartbeatPeriod)
		defer t.Stop()
		for {
			select {
			case <-pingStop:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				_ = wsConn.SetWriteDeadline(time.Now().Add(execPingWriteTimeout))
				if pingCodec.Send(wsConn, nil) != nil {
					// clear deadline before exit so wsConn.Close() in the receive goroutine doesn't hit an expired write deadline
					_ = wsConn.SetWriteDeadline(time.Time{})
					return
				}
				_ = wsConn.SetWriteDeadline(time.Time{})
			}
		}
	}()

	go func() {
		defer utilrecover.Panic(log)
		defer func() {
			// panic sentinel: len(errCh)==0 means the goroutine panicked before sending
			if len(errCh) == 0 {
				errCh <- errors.New("exec goroutine panicked")
			}
		}()
		defer close(frameCh)
		defer wsConn.Close()
		for {
			var msg []byte
			if recvErr := websocket.Message.Receive(wsConn, &msg); recvErr != nil {
				if netErr, ok := recvErr.(net.Error); ok && netErr.Timeout() {
					errCh <- errors.New("exec connection timed out: no response from server")
				} else if errors.Is(recvErr, io.EOF) {
					errCh <- errors.New("connection closed before exit-status frame")
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
				case <-pingStop:
					// caller returned; nobody is draining frameCh
					errCh <- errors.New("exec stream closed")
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
						statusMsg := status.Message
						if statusMsg == "" {
							statusMsg = "exec command failed with no message from server"
						}
						errCh <- errors.New(statusMsg)
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

// Caller owns both returned conns; closing tlsConn unblocks wsConn.Receive.
// TODO: replace with remotecommand.NewWebSocketExecutor when client-go >= v0.28.
func dialExecWebSocket(ctx context.Context, rc *restclient.Config, execURL *url.URL) (*websocket.Conn, *tls.Conn, error) {
	tlsConfig, err := restclient.TLSConfigFor(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("building TLS config: %w", err)
	}

	// rc.Proxy is not honoured; ARO clusters use rc.Dial for private-endpoint routing.
	dialAddr := execURL.Host
	if execURL.Port() == "" {
		dialAddr = net.JoinHostPort(execURL.Hostname(), "443")
	}
	var rawConn net.Conn
	if rc.Dial != nil {
		rawConn, err = rc.Dial(ctx, "tcp", dialAddr)
	} else {
		rawConn, err = (&net.Dialer{}).DialContext(ctx, "tcp", dialAddr)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("dial: %w", err)
	}

	tlsConf := tlsConfig
	if tlsConf == nil {
		tlsConf = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if tlsConf.ServerName == "" {
		tlsConf = tlsConf.Clone()
		tlsConf.ServerName = execURL.Hostname()
	}

	tlsConn := tls.Client(rawConn, tlsConf)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		rawConn.Close()
		return nil, nil, fmt.Errorf("TLS handshake: %w", err)
	}

	wsURL := *execURL
	wsURL.Scheme = "wss"
	originURL := url.URL{Scheme: "https", Host: execURL.Host}
	wsConfig := &websocket.Config{
		Location: &wsURL,
		Origin:   &originURL,
		Protocol: []string{"v4.channel.k8s.io"},
		Version:  websocket.ProtocolVersionHybi13,
		Header:   make(http.Header),
	}
	// bearer tokens and ExecProvider are silently ignored; auth uses TLS client cert from rc.TLSClientConfig
	hbConn := &heartbeatConn{Conn: tlsConn, deadline: execHeartbeatDeadline}
	if err := tlsConn.SetReadDeadline(time.Now().Add(execHeartbeatDeadline)); err != nil {
		tlsConn.Close()
		return nil, nil, fmt.Errorf("setting read deadline: %w", err)
	}

	rec := &recorderConn{Conn: hbConn}
	wsConn, err := websocket.NewClient(wsConfig, rec)
	if err != nil {
		tlsConn.Close()
		// NewClient reads headers but not the body; parse the recorder's buffer for a descriptive error
		if resp, respErr := http.ReadResponse(bufio.NewReader(bytes.NewReader(rec.buf[:rec.n])), nil); respErr == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var status metav1.Status
			if json.Unmarshal(body, &status) == nil && status.Message != "" {
				return nil, nil, errors.New(status.Message)
			}
			if bodyStr := strings.TrimSpace(string(body)); bodyStr != "" {
				if len(bodyStr) > 256 {
					bodyStr = bodyStr[:256] + "..."
				}
				return nil, nil, fmt.Errorf("WebSocket upgrade: HTTP %d: %s", resp.StatusCode, bodyStr)
			}
		}
		return nil, nil, fmt.Errorf("WebSocket upgrade: %w", err)
	}

	return wsConn, tlsConn, nil
}

// must match execOutputLimit; +1 lets the server stream one byte past the cap so limitedWriter detects truncation.
// TODO: share as a const with pkg/frontend/common.go if packages are refactored.
const podLogFollowLimit int64 = 1<<20 + 1

// KubeFollowPodLogs streams pod container logs to w.
func (k *kubeActions) KubeFollowPodLogs(ctx context.Context, namespace, podName, containerName string, w io.Writer) error {
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		LimitBytes: pointerutils.ToPtr(podLogFollowLimit),
	}
	stream, err := k.kubecli.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return fmt.Errorf("opening log stream for %s/%s/%s: %w", namespace, podName, containerName, err)
	}
	defer stream.Close()
	_, err = io.Copy(w, stream)
	if err != nil {
		return fmt.Errorf("streaming logs for %s/%s/%s: %w", namespace, podName, containerName, err)
	}
	return nil
}

func (k *kubeActions) ResolveGVR(groupKind string, optionalVersion string) (schema.GroupVersionResource, error) {
	return k.mapper.ResourceFor(schema.ParseGroupResource(groupKind).WithVersion(optionalVersion))
}

func (k *kubeActions) KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error) {
	gvr, err := k.ResolveGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	un, err := k.dyn.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

// KubeList lists resources. Pass optional label selectors to filter results (e.g., "app=foo", "env=prod").
func (k *kubeActions) KubeList(ctx context.Context, groupKind, namespace string, labelSelector ...string) ([]byte, error) {
	gvr, err := k.ResolveGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	selector := strings.Join(labelSelector, ",")

	// protect RP memory by not reading in more than 1000 items
	ul, err := k.dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{
		Limit:         1000,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	if ul.GetContinue() != "" {
		return nil, api.NewCloudError(
			http.StatusInternalServerError, api.CloudErrorCodeInternalServerError,
			groupKind, "Too many items returned.")
	}

	return ul.MarshalJSON()
}

func (k *kubeActions) KubeCreateOrUpdate(ctx context.Context, o *unstructured.Unstructured) error {
	gvr, err := k.ResolveGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	_, err = k.dyn.Resource(gvr).Namespace(o.GetNamespace()).Update(ctx, o, metav1.UpdateOptions{})
	if !kerrors.IsNotFound(err) {
		return err
	}

	_, err = k.dyn.Resource(gvr).Namespace(o.GetNamespace()).Create(ctx, o, metav1.CreateOptions{})
	return err
}

// Callers must ensure o.GetLabels()[labelKey] is a valid, non-empty label value.
func (k *kubeActions) KubeWatch(ctx context.Context, o *unstructured.Unstructured, labelKey string) (watch.Interface, error) {
	gvr, err := k.ResolveGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return nil, err
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelKey, o.GetLabels()[labelKey]),
	}

	w, err := k.dyn.Resource(gvr).Namespace(o.GetNamespace()).Watch(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (k *kubeActions) KubeDelete(ctx context.Context, groupKind, namespace, name string, force bool, propagationPolicy *metav1.DeletionPropagation) error {
	gvr, err := k.ResolveGVR(groupKind, "")
	if err != nil {
		return err
	}

	resourceDeleteOptions := metav1.DeleteOptions{}
	if force {
		resourceDeleteOptions.GracePeriodSeconds = pointerutils.ToPtr(int64(0))
	}

	if propagationPolicy != nil {
		resourceDeleteOptions.PropagationPolicy = propagationPolicy
	}

	return k.dyn.Resource(gvr).Namespace(namespace).Delete(ctx, name, resourceDeleteOptions)
}

func (k *kubeActions) CheckAPIServerReadyz(ctx context.Context) error {
	_, err := k.kubecli.Discovery().RESTClient().Get().AbsPath("/readyz").Do(ctx).Raw()
	if err != nil {
		return fmt.Errorf("API server readyz check failed: %w", err)
	}
	return nil
}
