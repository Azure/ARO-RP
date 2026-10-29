package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/installer/inputs"
)

type installerLogResult struct {
	forwarded bool
	err       error
}

func (m *manager) streamLogs(ctx context.Context, installerInputs *inputs.InstallerInputs) (bool, error) {
	var podName string
	err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		pods, err := m.client.CoreV1().Pods(installerInputs.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "job-name=" + installerInputs.JobName,
		})
		if err != nil {
			return false, fmt.Errorf("failed to list installer pods: %w", err)
		}
		for _, pod := range pods.Items {
			for _, status := range pod.Status.ContainerStatuses {
				if status.Name == "installer" && (status.State.Running != nil || status.State.Terminated != nil) {
					podName = pod.Name
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return false, err
	}

	stream, err := m.client.CoreV1().Pods(installerInputs.Namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "installer",
		Follow:    true,
	}).Stream(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to open installer log stream: %w", err)
	}
	defer stream.Close()

	log := m.log.WithFields(logrus.Fields{
		"clusterUUID": installerInputs.ClusterUUID,
		"executionID": installerInputs.ExecutionID,
	})
	buffer := newInstallerLogBuffer(func(logs string) {
		for _, line := range strings.Split(strings.TrimSuffix(logs, "\n"), "\n") {
			log.Infof("installer: %s", line)
		}
	})
	defer buffer.flush()

	// ReadSlice bounds individual reads even if the installer emits no newline.
	reader := bufio.NewReaderSize(stream, 32*1024)
	for {
		line, err := reader.ReadSlice('\n')
		buffer.add(line)
		switch {
		case err == nil, errors.Is(err, bufio.ErrBufferFull):
		case errors.Is(err, io.EOF):
			return buffer.received, nil
		default:
			return buffer.received, fmt.Errorf("failed to read installer log stream: %w", err)
		}
	}
}

const installerLogTruncationMarker = "\n... installer logs truncated ...\n"

// Forward the head immediately and retain a bounded tail for stream shutdown.
// This preserves timeout diagnostics without unbounded log output or buffering.
type installerLogBuffer struct {
	forward       func(string)
	headRemaining int
	tail          []byte
	truncated     bool
	received      bool
}

func newInstallerLogBuffer(forward func(string)) *installerLogBuffer {
	head := (maxInstallerLogBytes - len(installerLogTruncationMarker)) / 2
	return &installerLogBuffer{
		forward:       forward,
		headRemaining: head,
		tail:          make([]byte, 0, maxInstallerLogBytes-head),
	}
}

func (b *installerLogBuffer) add(data []byte) {
	if len(data) == 0 {
		return
	}
	b.received = true
	n := min(b.headRemaining, len(data))
	if n > 0 {
		b.forward(string(data[:n]))
		b.headRemaining -= n
		data = data[n:]
	}
	if len(data) >= cap(b.tail) {
		b.truncated = b.truncated || len(b.tail)+len(data) > cap(b.tail)
		b.tail = append(b.tail[:0], data[len(data)-cap(b.tail):]...)
		return
	}
	if excess := len(b.tail) + len(data) - cap(b.tail); excess > 0 {
		copy(b.tail, b.tail[excess:])
		b.tail = b.tail[:len(b.tail)-excess]
		b.truncated = true
	}
	b.tail = append(b.tail, data...)
}

func (b *installerLogBuffer) flush() {
	if b.truncated {
		b.forward(installerLogTruncationMarker)
		b.tail = b.tail[len(installerLogTruncationMarker):]
	}
	if len(b.tail) > 0 {
		b.forward(string(b.tail))
	}
}
