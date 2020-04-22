package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	mockportforward "github.com/Azure/ARO-RP/pkg/util/mocks/portforward"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestWaitForMustGatherPod(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name      string
		run       bool
		wantError string
	}{
		{
			name:      "timeout",
			wantError: "timed out waiting for the condition",
		},
		{
			name: "success",
			run:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ka := &kubeactions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				oc:  &api.OpenShiftCluster{},
				cli: fake.NewSimpleClientset(),
			}

			ns, err := createMustGatherNamespace(ka)
			if err != nil {
				t.Fatal(err)
			}

			pod, err := createMustGatherPod(ka, ns)
			if err != nil {
				t.Fatal(err)
			}

			if tt.run {
				pod.Status = corev1.PodStatus{
					Phase: corev1.PodRunning,
				}
				ka.cli.CoreV1().Pods(pod.Namespace).Update(pod)
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			err = waitForMustGatherPod(timeoutCtx, ka, pod)

			if err != nil && err.Error() != tt.wantError {
				t.Fatal(err)
			}
		})
	}
}

func TestCopyMustGatherLogs(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name      string
		mocks     func(e *mockportforward.MockExec, ctx context.Context, pod *corev1.Pod, rc io.ReadCloser)
		wantError string
	}{
		{
			name: "failure",
			mocks: func(e *mockportforward.MockExec, ctx context.Context, pod *corev1.Pod, rc io.ReadCloser) {
				e.EXPECT().
					Stdout(ctx, pod.Namespace, pod.Name, "copy", []string{"tar", "cz", "/must-gather"}).
					Return(nil, errors.New("no"))
			},
			wantError: "no",
		},
		{
			name: "success",
			mocks: func(e *mockportforward.MockExec, ctx context.Context, pod *corev1.Pod, rc io.ReadCloser) {
				e.EXPECT().
					Stdout(ctx, pod.Namespace, pod.Name, "copy", []string{"tar", "cz", "/must-gather"}).
					Return(rc, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ka := &kubeactions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				cli: fake.NewSimpleClientset(),
			}

			ns, err := createMustGatherNamespace(ka)
			if err != nil {
				t.Fatal(err)
			}

			pod, err := createMustGatherPod(ka, ns)
			if err != nil {
				t.Fatal(err)
			}

			var w bytes.Buffer

			controller := gomock.NewController(t)
			defer controller.Finish()

			timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			e := mockportforward.NewMockExec(controller)
			rc := ioutil.NopCloser(strings.NewReader("all pods are doing just fine 🙂"))
			tt.mocks(e, timeoutCtx, pod, rc)

			err = copyMustGatherLogs(timeoutCtx, ka, pod, &w, e)

			if err != nil && err.Error() != tt.wantError {
				t.Fatal(err)
			}
		})
	}
}
