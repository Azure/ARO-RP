package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
)

type Manager interface {
	Get(ctx context.Context, resourceGroup string, resourceName string) (*v1.Config, error)
	Print(ctx context.Context, resourceGroup string, resourceName string) error
}

type manager struct {
	log               *logrus.Entry
	openshiftclusters redhatopenshift.OpenShiftClustersClient
}

func NewManager(log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) Manager {
	return &manager{
		log:               log,
		openshiftclusters: redhatopenshift.NewOpenShiftClustersClient(subscriptionID, authorizer),
	}
}

func (m *manager) Get(ctx context.Context, resourceGroup, resourceName string) (*v1.Config, error) {
	return m.get(ctx, resourceGroup, resourceName)
}

func (m *manager) Print(ctx context.Context, resourceGroup, resourceName string) error {
	adminKubeconfig, err := m.get(ctx, resourceGroup, resourceName)
	if err != nil {
		return err
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "    ")
	return e.Encode(adminKubeconfig)
}

func (m *manager) get(ctx context.Context, resourceGroup, resourceName string) (*v1.Config, error) {
	oc, err := m.openshiftclusters.Get(ctx, resourceGroup, resourceName)
	if err != nil {
		return nil, err
	}

	creds, err := m.openshiftclusters.ListCredentials(ctx, resourceGroup, resourceName)
	if err != nil {
		return nil, err
	}
	tokenURL, err := getTokenURLFromConsoleURL(*oc.OpenShiftClusterProperties.ConsoleProfile.URL)
	if err != nil {
		return nil, err
	}

	var token string

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		token, err = getAuthorizedToken(ctx, tokenURL, *creds.KubeadminUsername, *creds.KubeadminPassword)
		if err != nil {
			log.Print(err)
			return false, nil
		}

		return true, nil
	}, timeoutCtx.Done())
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(*oc.OpenShiftClusterProperties.ApiserverProfile.URL)
	if err != nil {
		return nil, err
	}

	return makeKubeconfig(u.Host, *creds.KubeadminUsername, token, "kube-system"), nil
}
