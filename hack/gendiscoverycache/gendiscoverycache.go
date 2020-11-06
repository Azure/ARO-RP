package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/tools/clientcmd"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
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

	cvClient, err := configclient.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	clusterVersion, err := getClusterVersion(ctx, cvClient)
	if err != nil {
		return err
	}

	err = os.RemoveAll(discoveryCacheDir)
	if err != nil {
		return err
	}

	cli, err := disk.NewCachedDiscoveryClientForConfig(restconfig, discoveryCacheDir, "", 0)
	if err != nil {
		return err
	}

	return writeAssets(cli, clusterVersion, discoveryCacheDir)
}

func getClusterVersion(ctx context.Context, cvClient configclient.Interface) (string, error) {
	cv, err := cvClient.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return history.Version, nil
		}
	}

	// Should never happen as a successfully created cluster
	// should have at least one completed update.
	return "", errors.New("could find actual cluster version")
}

func writeAssets(cli discovery.DiscoveryInterface, clusterVersion, cacheDir string) error {
	_, _, err := cli.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	versionPath := filepath.Join(cacheDir, "assets_version")
	err = ioutil.WriteFile(versionPath, []byte(clusterVersion+"\n"), 0666)
	if err != nil {
		return err
	}

	return canonicalizeAssets(cacheDir)
}

func canonicalizeAssets(cacheDir string) error {
	return filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		switch filepath.Base(path) {
		case "servergroups.json":
			return formatServerGroups(path)
		case "serverresources.json":
			return canonicalizeServerResources(path)
		}

		return nil
	})
}

func formatServerGroups(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var i interface{}

	err = json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	b, err = json.MarshalIndent(i, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, append(b, '\n'), 0666)
}

func canonicalizeServerResources(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var l *metav1.APIResourceList

	err = json.Unmarshal(b, &l)
	if err != nil {
		return err
	}

	sort.Slice(l.APIResources, func(i, j int) bool {
		return strings.Compare(l.APIResources[i].Name, l.APIResources[j].Name) < 0
	})

	for _, r := range l.APIResources {
		sort.Strings(r.Categories)
		sort.Strings(r.ShortNames)
		sort.Strings(r.Verbs)
	}

	b, err = json.MarshalIndent(l, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, append(b, '\n'), 0666)
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
