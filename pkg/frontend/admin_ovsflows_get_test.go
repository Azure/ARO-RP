package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	fakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestCreateOVSFlowPod(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	client := fakeclient.NewSimpleClientset()

	var c Client = Client{Clientset: client}

	_, err := c.createOVSFlowPod(ctx, "fakenode", log)
	if err != nil {
		t.Error(err)
	}

	// Testing the function if a pod with same name already exists.
	_, err = c.createOVSFlowPod(ctx, "fakenode", log)
	if err != nil {
		t.Error(err)
	}

}

func TestGetPodLogs(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	client := fakeclient.NewSimpleClientset()

	var c Client = Client{Clientset: client}
	pod, _ := c.createOVSFlowPod(ctx, "fakenode", log)

	_, err := c.getPodLogs(ctx, log, pod)
	if err != nil {
		t.Error(err)
	}

}
