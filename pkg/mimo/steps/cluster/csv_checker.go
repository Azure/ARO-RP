package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

//go:embed concerning_csvs/base.txt
var concerningCSVsBase []byte

//go:embed concerning_csvs/*.diff.txt
var concerningCSVsDiffs embed.FS

var csvGVK = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Version: "v1alpha1",
	Kind:    "ClusterServiceVersionList",
}

// DetectConcerningClusterServiceVersions checks for Red Hat Operators that
// were inadvertently upgraded to 4.18 catalog versions on 4.12-4.17 clusters
// due to an incorrect catalog content release on 2026-02-03.
// https://access.redhat.com/solutions/7137887
func DetectConcerningClusterServiceVersions(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return err
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}

	cv := &configv1.ClusterVersion{}
	err = ch.GetOne(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("unable to get ClusterVersion: %w", err))
	}

	if len(cv.Status.History) == 0 {
		return mimo.TerminalError(fmt.Errorf("ClusterVersion has no update history"))
	}

	clusterVersion := cv.Status.History[0].Version
	majorMinor, err := parseMajorMinor(clusterVersion)
	if err != nil {
		return mimo.TerminalError(err)
	}

	concerningSet, err := loadConcerningCSVs(majorMinor)
	if err != nil {
		th.SetResultMessage(fmt.Sprintf("no concerning CSV data for version %s; skipping check", majorMinor))
		return nil
	}

	csvList := &unstructured.UnstructuredList{}
	csvList.SetGroupVersionKind(csvGVK)
	err = ch.List(ctx, csvList)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("unable to list ClusterServiceVersions: %w", err))
	}

	seen := make(map[string]bool)
	for i := range csvList.Items {
		name := csvList.Items[i].GetName()
		seen[name] = true
	}

	var found []string
	for name := range seen {
		if concerningSet[name] {
			found = append(found, name)
		}
	}
	sort.Strings(found)

	if len(found) == 0 {
		th.SetResultMessage(fmt.Sprintf(
			"no inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running %s",
			clusterVersion,
		))
		return nil
	}

	return mimo.TerminalError(fmt.Errorf(
		"inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running %s:\n%s",
		clusterVersion, strings.Join(found, "\n"),
	))
}

// parseMajorMinor extracts "X.Y" from a version string like "X.Y.Z".
func parseMajorMinor(version string) (string, error) {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("unable to parse major.minor from version %q", version)
	}
	return parts[0] + "." + parts[1], nil
}

// loadConcerningCSVs builds the set of concerning CSV names for the given
// major.minor version by loading the shared base list and applying the
// version-specific diff (additions prefixed with "+", removals with "-").
func loadConcerningCSVs(majorMinor string) (map[string]bool, error) {
	diffData, err := concerningCSVsDiffs.ReadFile(fmt.Sprintf("concerning_csvs/%s.diff.txt", majorMinor))
	if err != nil {
		return nil, fmt.Errorf("no concerning CSV data for version %s", majorMinor)
	}

	result := loadLines(concerningCSVsBase)

	scanner := bufio.NewScanner(strings.NewReader(string(diffData)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case '+':
			result[line[1:]] = true
		case '-':
			delete(result, line[1:])
		}
	}
	return result, nil
}

// loadLines parses raw bytes into a set of non-empty trimmed lines.
func loadLines(data []byte) map[string]bool {
	result := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			result[line] = true
		}
	}
	return result
}
