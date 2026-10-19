package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/installer/inputs"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestInstallerLogBuffer(t *testing.T) {
	for _, size := range []int{0, 1, maxInstallerLogBytes / 2, maxInstallerLogBytes - 1, maxInstallerLogBytes, maxInstallerLogBytes + 1, 3 * maxInstallerLogBytes} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			logs := strings.Repeat("0123456789abcdef\n", size/17+1)[:size]
			var output strings.Builder
			buffer := newInstallerLogBuffer(func(s string) { output.WriteString(s) })
			for offset := 0; offset < len(logs); offset += 32 * 1024 {
				buffer.add([]byte(logs[offset:min(offset+32*1024, len(logs))]))
			}
			if len(logs) > 0 && output.Len() == 0 {
				t.Fatal("head logs were not forwarded before shutdown")
			}
			if len(buffer.tail) > maxInstallerLogBytes {
				t.Fatal("tail buffer exceeded the log limit")
			}
			buffer.flush()
			if got, want := output.String(), truncateInstallerLogs(logs); got != want {
				t.Fatalf("forwarded logs differ from bounded head/tail output: got %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

func TestInstallForwardsLogsBeforeTimedOutPodDisappears(t *testing.T) {
	for _, tt := range []struct {
		name         string
		brokenStream bool
	}{
		{name: "pod disappears at deadline"},
		{name: "broken stream falls back to snapshot", brokenStream: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const jobName = "installer-execution-id"
			var (
				podDeleted       atomic.Bool
				snapshotRequests atomic.Int32
			)
			streamClosed := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var result interface{}
				switch {
				case r.URL.Path == "/api/v1/namespaces/"+Namespace:
					result = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: Namespace}}
				case strings.HasSuffix(r.URL.Path, "/serviceaccounts/"+ServiceAccountName):
					result = &corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name: ServiceAccountName,
							Annotations: map[string]string{
								"azure.workload.identity/client-id": "identity-client-id",
							},
						},
					}
				case strings.HasSuffix(r.URL.Path, "/secrets") && r.Method == http.MethodPost:
					secret := &corev1.Secret{}
					if err := json.NewDecoder(r.Body).Decode(secret); err != nil {
						t.Errorf("decode secret: %v", err)
						http.Error(w, "invalid secret", http.StatusBadRequest)
						return
					}
					result = secret
				case strings.Contains(r.URL.Path, "/secrets/") && r.Method == http.MethodGet:
					http.NotFound(w, r)
					return
				case strings.HasSuffix(r.URL.Path, "/jobs/"+jobName):
					result = &batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name: jobName,
							Labels: map[string]string{
								"execution-id":     "execution-id",
								"aro-cluster-uuid": "cluster-uuid",
							},
						},
						Status: batchv1.JobStatus{Active: 1},
					}
				case strings.HasSuffix(r.URL.Path, "/pods"):
					pods := &corev1.PodList{}
					if !podDeleted.Load() {
						pods.Items = []corev1.Pod{{
							ObjectMeta: metav1.ObjectMeta{Name: "installer-pod"},
							Status: corev1.PodStatus{
								ContainerStatuses: []corev1.ContainerStatus{{
									Name:  "installer",
									State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
								}},
							},
						}}
					}
					result = pods
				case strings.HasSuffix(r.URL.Path, "/pods/installer-pod/log"):
					if r.URL.Query().Get("follow") != "true" {
						snapshotRequests.Add(1)
						if tt.brokenStream {
							fmt.Fprintln(w, "snapshot after stream failure")
							return
						}
						http.Error(w, "pod was deleted", http.StatusNotFound)
						return
					}
					if r.URL.Query().Get("container") != "installer" {
						t.Error("log stream did not select the installer container")
					}
					w.Header().Set("Content-Type", "text/plain")
					if tt.brokenStream {
						w.Header().Set("Content-Length", "1000")
						fmt.Fprintln(w, "SHIM_PASS_INPUTS before timeout")
						close(streamClosed)
						return
					}
					fmt.Fprintln(w, "SHIM_PASS_INPUTS before timeout")
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					close(streamClosed)
					return
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.NotFound(w, r)
					return
				}
				if err := json.NewEncoder(w).Encode(result); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer server.Close()
			client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			hook, log := testlog.New()
			manager := newManager(log, client)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- manager.Install(ctx, &inputs.InstallerInputs{
					Namespace:                 Namespace,
					JobName:                   jobName,
					ExecutionID:               "execution-id",
					ClusterUUID:               "cluster-uuid",
					InstallerIdentityClientID: "identity-client-id",
					PullSecretJSON:            []byte(`{"auths":{}}`),
					Timeout:                   2 * time.Second,
				})
			}()
			err = wait.PollUntilContextCancel(ctx, 10*time.Millisecond, true, func(context.Context) (bool, error) {
				for _, entry := range hook.AllEntries() {
					if entry.Message == "installer: SHIM_PASS_INPUTS before timeout" {
						if entry.Data["clusterUUID"] != "cluster-uuid" || entry.Data["executionID"] != "execution-id" {
							return false, fmt.Errorf("installer log is missing execution fields: %v", entry.Data)
						}
						return true, nil
					}
				}
				return false, nil
			})
			if err != nil {
				t.Fatalf("installer diagnostics were not forwarded while the Job ran: %v", err)
			}
			podDeleted.Store(!tt.brokenStream)
			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("Install returned %v, want timeout", err)
				}
			case <-ctx.Done():
				t.Fatal("Install did not stop at its deadline")
			}
			select {
			case <-streamClosed:
			case <-ctx.Done():
				t.Fatal("log stream leaked after Install returned")
			}
			if tt.brokenStream {
				if got := snapshotRequests.Load(); got != 1 {
					t.Fatalf("attempted %d fallback snapshots, want 1", got)
				}
				var warned, snapshotted bool
				for _, entry := range hook.AllEntries() {
					warned = warned || entry.Message == "failed to stream installer logs; logs may be incomplete"
					snapshotted = snapshotted || entry.Message == "installer: snapshot after stream failure"
				}
				if !warned || !snapshotted {
					t.Fatalf("stream failure warning = %v, fallback logs = %v", warned, snapshotted)
				}
			} else if got := snapshotRequests.Load(); got != 0 {
				t.Fatalf("attempted %d post-timeout log snapshots", got)
			}
		})
	}
}

func TestStreamInstallerLogs(t *testing.T) {
	for _, tt := range []struct {
		name          string
		waiting       bool
		terminated    bool
		listError     bool
		logError      bool
		brokenStream  bool
		logs          string
		wantForwarded bool
		wantError     string
	}{
		{name: "running", logs: "first\nsecond\n", wantForwarded: true},
		{name: "terminated", terminated: true, logs: "last line without newline", wantForwarded: true},
		{name: "empty"},
		{name: "waiting for container", waiting: true, wantError: "context deadline exceeded"},
		{name: "list error", listError: true, wantError: "failed to list installer pods"},
		{name: "log permission denied", logError: true, wantError: "failed to open installer log stream"},
		{name: "partial read failure", brokenStream: true, logs: "partial\n", wantForwarded: true, wantError: "failed to read installer log stream"},
		{name: "long line", logs: strings.Repeat("a", 2*maxInstallerLogBytes) + "tail\n", wantForwarded: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/pods"):
					if tt.listError {
						http.Error(w, "list failed", http.StatusForbidden)
						return
					}
					state := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
					if tt.waiting {
						state = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}
					}
					if tt.terminated {
						state = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
					}
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(&corev1.PodList{
						Items: []corev1.Pod{{
							ObjectMeta: metav1.ObjectMeta{Name: "installer-pod"},
							Status: corev1.PodStatus{
								ContainerStatuses: []corev1.ContainerStatus{{Name: "installer", State: state}},
							},
						}},
					}); err != nil {
						t.Errorf("encode pod list: %v", err)
					}
				case strings.HasSuffix(r.URL.Path, "/log"):
					if tt.waiting {
						t.Error("attempted logs before the installer container started")
					}
					if tt.logError {
						http.Error(w, "log access denied", http.StatusForbidden)
						return
					}
					if tt.brokenStream {
						w.Header().Set("Content-Length", fmt.Sprint(len(tt.logs)+1))
					}
					fmt.Fprint(w, tt.logs)
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			hook, log := testlog.New()
			timeout := 5 * time.Second
			if tt.waiting {
				timeout = 100 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			forwarded, err := newManager(log, client).streamLogs(ctx, &inputs.InstallerInputs{
				Namespace: Namespace, JobName: "installer-job",
			})
			if forwarded != tt.wantForwarded {
				t.Errorf("forwarded = %v, want %v", forwarded, tt.wantForwarded)
			}
			if tt.wantError == "" && err != nil || tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Errorf("stream error = %v, want %q", err, tt.wantError)
			}
			if tt.wantForwarded && len(hook.AllEntries()) == 0 {
				t.Error("stream data was not forwarded to the log pipeline")
			}
			if tt.name == "long line" {
				entries := hook.AllEntries()
				if got := entries[len(entries)-1].Message; !strings.HasSuffix(got, "tail") {
					t.Error("bounded logs lost the tail of a long line")
				}
			}
		})
	}
}
