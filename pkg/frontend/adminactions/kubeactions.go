package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

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
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// execFrame carries a single v4.channel.k8s.io data frame from the inner
// goroutine of KubeExecStream to the main goroutine for writing.
type execFrame struct {
	channelID byte
	data      []byte
}

// KubeActions are those that involve k8s objects, and thus depend upon k8s clients being createable
type KubeActions interface {
	KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	KubeList(ctx context.Context, groupKind, namespace string) ([]byte, error)
	KubeCreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	KubeDelete(ctx context.Context, groupKind, namespace, name string, force bool, propagationPolicy *metav1.DeletionPropagation) error
	// KubeExecStream execs command in pod/container, streaming stdout and stderr to
	// the provided writers. command is passed directly to the container runtime;
	// wrap in []string{"sh", "-c", cmd} for shell features.
	//
	// Returns ctx.Err() on context cancellation (after the inner goroutine has
	// exited), a write error if a caller-supplied writer fails, or an error from
	// the command's exit-status frame.
	KubeExecStream(ctx context.Context, namespace, pod, container string, command []string, stdout, stderr io.Writer) error
	// KubeFollowPodLogs streams the logs of a pod container to w. If containerName
	// is empty, Kubernetes' default log selection applies and may fail when the
	// pod has multiple containers.
	KubeFollowPodLogs(ctx context.Context, namespace, podName, containerName string, w io.Writer) error
	ResolveGVR(groupKind string, optionalVersion string) (schema.GroupVersionResource, error)
	CordonNode(ctx context.Context, nodeName string, unschedulable bool) error
	DrainNode(ctx context.Context, nodeName string) error
	ApproveCsr(ctx context.Context, csrName string) error
	ApproveAllCsrs(ctx context.Context) error
	KubeGetPodLogs(ctx context.Context, namespace, name, containerName string) ([]byte, error)
	// KubeWatch returns a watch for objects matching the label key=value extracted from o.GetLabels()[label].
	KubeWatch(ctx context.Context, o *unstructured.Unstructured, label string) (watch.Interface, error)
	// Fetch top pods and nodes metrics
	TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]PodMetrics, error)
	TopNodes(ctx context.Context, restConfig *restclient.Config) ([]NodeMetrics, error)
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

// checkContainerReady verifies the named container in the pod is ready to
// accept exec connections. This is a pre-flight check: the WebSocket upgrade
// returns an opaque "bad status" error when the API server cannot proxy the
// exec request to the kubelet (e.g. because the container is still starting),
// so we surface a descriptive error before attempting the connection.
func (k *kubeActions) checkContainerReady(ctx context.Context, namespace, podName, container string) error {
	p, err := k.kubecli.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting pod %s/%s: %w", namespace, podName, err)
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Name != container {
			continue
		}
		if cs.Ready {
			return nil
		}
		switch {
		case cs.State.Waiting != nil:
			return fmt.Errorf("container %q in pod %s/%s is not ready: waiting (%s: %s)",
				container, namespace, podName, cs.State.Waiting.Reason, cs.State.Waiting.Message)
		case cs.State.Terminated != nil:
			return fmt.Errorf("container %q in pod %s/%s is not ready: terminated (exit code %d: %s)",
				container, namespace, podName, cs.State.Terminated.ExitCode, cs.State.Terminated.Reason)
		default:
			return fmt.Errorf("container %q in pod %s/%s is not ready", container, namespace, podName)
		}
	}
	return fmt.Errorf("container %q not found in pod %s/%s status (pod phase: %s)",
		container, namespace, podName, p.Status.Phase)
}

func (k *kubeActions) KubeExecStream(ctx context.Context, namespace, pod, container string, command []string, stdout, stderr io.Writer) error {
	if err := k.checkContainerReady(ctx, namespace, pod, container); err != nil {
		return err
	}

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
	// sync.Once makes closeTLS idempotent: called explicitly on cancellation and by defer.
	var closeOnce sync.Once
	closeTLS := func() { closeOnce.Do(func() { tlsConn.Close() }) }
	defer closeTLS()

	// v4.channel.k8s.io: ch1=stdout, ch2=stderr, ch3=exit status (metav1.Status JSON)
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

// dialExecWebSocket dials the API server and upgrades to a v4.channel.k8s.io WebSocket exec connection.
// Caller owns both wsConn and tlsConn; closing tlsConn unblocks pending Receive on wsConn.
// TODO(ARO-23146): replace with remotecommand.NewWebSocketExecutor when client-go >= v0.28.
func dialExecWebSocket(ctx context.Context, rc *restclient.Config, execURL *url.URL) (*websocket.Conn, *tls.Conn, error) {
	tlsConfig, err := restclient.TLSConfigFor(rc)
	if err != nil {
		return nil, nil, err
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
		tlsConf = &tls.Config{} //nolint:gosec
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
	// rc.ExecProvider and rc.AuthProvider are not supported; ARO clusters use bearer-token or client-cert.
	bearerToken := rc.BearerToken
	if bearerToken == "" && rc.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(rc.BearerTokenFile)
		if err != nil {
			tlsConn.Close()
			return nil, nil, fmt.Errorf("reading bearer token file: %w", err)
		}
		bearerToken = strings.TrimSpace(string(tokenBytes))
	}
	if bearerToken != "" {
		wsConfig.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	wsConn, err := websocket.NewClient(wsConfig, tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, nil, fmt.Errorf("WebSocket upgrade: %w", err)
	}

	return wsConn, tlsConn, nil
}

func (k *kubeActions) KubeFollowPodLogs(ctx context.Context, namespace, podName, containerName string, w io.Writer) error {
	opts := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
	}
	stream, err := k.kubecli.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(w, stream)
	return err
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

func (k *kubeActions) KubeList(ctx context.Context, groupKind, namespace string) ([]byte, error) {
	gvr, err := k.ResolveGVR(groupKind, "")
	if err != nil {
		return nil, err
	}

	// protect RP memory by not reading in more than 1000 items
	ul, err := k.dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1000})
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

func (k *kubeActions) KubeWatch(ctx context.Context, o *unstructured.Unstructured, labelKey string) (watch.Interface, error) {
	gvr, err := k.ResolveGVR(o.GroupVersionKind().GroupKind().String(), o.GroupVersionKind().Version)
	if err != nil {
		return nil, err
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v", labelKey, o.GetLabels()[labelKey]),
		Watch:         true,
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
