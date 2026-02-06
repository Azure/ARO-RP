package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	configclient "github.com/openshift/client-go/config/clientset/versioned"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const discoveryCacheDir = "pkg/util/dynamichelper/discovery/cache"

func run(ctx context.Context, log *logrus.Entry) error {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return err
	}

	err = os.RemoveAll(discoveryCacheDir)
	if err != nil {
		return err
	}

	err = genDiscoveryCache(restconfig)
	if err != nil {
		return err
	}

	err = genRBAC(restconfig)
	if err != nil {
		return err
	}

	return writeVersion(ctx, restconfig)
}

func writeVersion(ctx context.Context, restconfig *rest.Config) error {
	configcli, err := configclient.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return err
	}
	clusterVersion, err := version.GetClusterVersion(cv)
	if err != nil {
		return err
	}

	versionPath := filepath.Join(discoveryCacheDir, "assets_version")
	return os.WriteFile(versionPath, []byte(clusterVersion.String()+"\n"), 0o666)
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
