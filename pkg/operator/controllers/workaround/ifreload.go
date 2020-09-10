package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (*ifReload) Ensure() error {
	return nil
}

func (i *ifReload) Remove() error {
	err := i.cli.CoreV1().Namespaces().Delete(kubeNamespace, nil)
	if !errors.IsNotFound(err) {
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
