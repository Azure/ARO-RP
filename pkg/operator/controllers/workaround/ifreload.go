package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

const kubeNamespace = "openshift-azure-ifreload"

type ifReload struct {
	log          *logrus.Entry
	cli          kubernetes.Interface
	versionFixed *version.Version
}

func (*ifReload) Name() string {
	return "ifReload"
}

func (i *ifReload) IsRequired(clusterVersion *version.Version) bool {
	return clusterVersion.Lt(i.versionFixed)
}

func (*ifReload) Ensure(ctx context.Context) error {
	return nil
}

func (i *ifReload) Remove(ctx context.Context) error {
	err := i.cli.CoreV1().Namespaces().Delete(ctx, kubeNamespace, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func NewIfReload(log *logrus.Entry, cli kubernetes.Interface) Workaround {
	verFixed, _ := version.ParseVersion("4.4.10")

	return &ifReload{
		log:          log,
		cli:          cli,
		versionFixed: verFixed,
	}
}
