package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/rest"
)

func genDiscoveryCache(restconfig *rest.Config) error {
	cli, err := disk.NewCachedDiscoveryClientForConfig(restconfig, discoveryCacheDir, "", 0)
	if err != nil {
		return err
	}

	_, _, err = cli.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	return canonicalizeAssets()
}

func canonicalizeAssets() error {
	return filepath.Walk(discoveryCacheDir, func(path string, info os.FileInfo, err error) error {
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
