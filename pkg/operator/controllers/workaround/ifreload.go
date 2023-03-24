package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type ifReload struct {
	log *logrus.Entry

	client client.Client

	versionFixed *version.Version
}

func NewIfReload(log *logrus.Entry, client client.Client) Workaround {
	verFixed, _ := version.ParseVersion("4.4.10")

	return &ifReload{
		log:          log,
		client:       client,
		versionFixed: verFixed,
	}
}

func (*ifReload) Name() string {
	return "ifReload"
}

func (i *ifReload) IsRequired(clusterVersion *version.Version, cluster *arov1alpha1.Cluster) bool {
	return clusterVersion.Lt(i.versionFixed)
}

func (i *ifReload) Ensure(ctx context.Context) error {
	i.log.Debug("ensure ifReload")
	return nil
}

func (i *ifReload) Remove(ctx context.Context) error {
	i.log.Debug("remove ifReload")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeNamespace,
		},
	}
	err := i.client.Delete(ctx, ns)
	if kerrors.IsNotFound(err) {
		return nil
	}
	return err
}
