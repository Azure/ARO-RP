package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

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
	// On context cancellation, returns ctx.Err() immediately. The underlying
	// receive goroutine may still write briefly to stdout/stderr; callers must
	// not read buffered writers when ctx.Err() != nil.
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
	// KubeWatch returns a watch object for the provided label selector key
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

// TODO: replace golang.org/x/net/websocket with remotecommand.NewWebSocketExecutor once
// the project upgrades to client-go v0.28 or newer.
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

	execURL := req.URL()

	tlsConfig, err := restclient.TLSConfigFor(k.rc)
	if err != nil {
		return err
	}

	// Dial through the cluster's private endpoint IP via restconfig.Dial if set.
	dialAddr := execURL.Host
	if execURL.Port() == "" {
		dialAddr = net.JoinHostPort(execURL.Hostname(), "443")
	}
	var rawConn net.Conn
	if k.rc.Dial != nil {
		rawConn, err = k.rc.Dial(ctx, "tcp", dialAddr)
	} else {
		rawConn, err = (&net.Dialer{}).DialContext(ctx, "tcp", dialAddr)
	}
	if err != nil {
		return fmt.Errorf("dial: %w", err)
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
		return fmt.Errorf("TLS handshake: %w", err)
	}

	// Upgrade to WebSocket with the Kubernetes exec subprotocol.
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
	// Bearer token auth; cert auth is handled by the TLS client certificate.
	if k.rc.BearerToken != "" {
		wsConfig.Header.Set("Authorization", "Bearer "+k.rc.BearerToken)
	}

	wsConn, err := websocket.NewClient(wsConfig, tlsConn)
	if err != nil {
		tlsConn.Close()
		return fmt.Errorf("WebSocket upgrade: %w", err)
	}

	// k8s v4.channel.k8s.io protocol: each frame = [channelID byte][data...]
	//   channel 1 = stdout, channel 2 = stderr, channel 3 = exit status (metav1.Status JSON)
	type result struct{ err error }
	resultCh := make(chan result, 1)

	go func() {
		defer wsConn.Close()
		receivedStatus := false
		for {
			var msg []byte
			if recvErr := websocket.Message.Receive(wsConn, &msg); recvErr != nil {
				if recvErr == io.EOF && receivedStatus {
					resultCh <- result{}
				} else if recvErr == io.EOF {
					resultCh <- result{err: fmt.Errorf("connection closed before exit-status frame")}
				} else {
					resultCh <- result{err: recvErr}
				}
				return
			}
			if len(msg) == 0 {
				continue
			}
			channelID, data := msg[0], msg[1:]
			switch channelID {
			case 1:
				if _, writeErr := stdout.Write(data); writeErr != nil {
					resultCh <- result{err: writeErr}
					return
				}
			case 2:
				if _, writeErr := stderr.Write(data); writeErr != nil {
					resultCh <- result{err: writeErr}
					return
				}
			case 3:
				// Exit status: a Failure status means the command returned non-zero.
				receivedStatus = true
				var status metav1.Status
				if len(data) > 0 {
					if jsonErr := json.Unmarshal(data, &status); jsonErr != nil {
						resultCh <- result{err: fmt.Errorf("malformed exit-status frame: %w", jsonErr)}
						return
					}
					if status.Status == metav1.StatusFailure {
						resultCh <- result{err: fmt.Errorf("command failed: %s", status.Message)}
						return
					}
				}
				resultCh <- result{}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		wsConn.Close() // unblock the read goroutine
		return ctx.Err()
	case r := <-resultCh:
		return r.err
	}
}

func (k *kubeActions) KubeFollowPodLogs(ctx context.Context, namespace, podName, containerName string, w io.Writer) error {
	opts := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
	}
	rc, err := k.kubecli.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(w, rc)
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
