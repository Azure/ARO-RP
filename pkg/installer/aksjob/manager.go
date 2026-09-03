package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/installer/inputs"
)

// Manager handles cluster installation using Kubernetes Jobs on AKS
type Manager interface {
	// Install creates and runs an installer Job, returning after it completes
	Install(ctx context.Context, inputs *inputs.InstallerInputs) error

	// GetLogs retrieves logs from the installer Job
	GetLogs(ctx context.Context, namespace, jobName string) (string, error)

	// Cleanup removes installer resources (namespace, Job, secrets)
	Cleanup(ctx context.Context, namespace string) error

	// GetStatus returns the current status of an installer Job
	GetStatus(ctx context.Context, namespace, jobName string) (*JobStatus, error)
}

// JobStatus represents the status of an installer Job
type JobStatus struct {
	Phase          JobPhase
	ExitCode       *int32
	StartTime      *string
	CompletionTime *string
	Message        string
}

// JobPhase represents the phase of a Job
type JobPhase string

const (
	JobPhaseUnknown   JobPhase = "Unknown"
	JobPhasePending   JobPhase = "Pending"
	JobPhaseRunning   JobPhase = "Running"
	JobPhaseSucceeded JobPhase = "Succeeded"
	JobPhaseFailed    JobPhase = "Failed"
	JobPhaseTimedOut  JobPhase = "TimedOut"
)

type manager struct {
	log    *logrus.Entry
	client kubernetes.Interface
	config *rest.Config
}

// New creates a new AKS Job installer manager
func New(log *logrus.Entry, config *rest.Config) (Manager, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &manager{
		log:    log,
		client: client,
		config: config,
	}, nil
}
