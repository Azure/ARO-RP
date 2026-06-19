package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

const (
	operatorImageRepository = "aro"
	operatorTagsPageSize    = 1000
)

var operatorImageVersionRegex = regexp.MustCompile(`^v(\d{8})\.(\d+)$`)

type operatorImageVersion struct {
	raw   string
	date  int
	build int
}

type acrTagsListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

var listACRRepositoryTagsFn = listACRRepositoryTags

func EnsureOperatorImageUpdateScheduled(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	oc := th.GetOpenShiftClusterDocument()
	props := oc.OpenShiftCluster.Properties
	profile := acrtoken.GetRegistryProfileFromSlice(th.Environment(), props.RegistryProfiles)
	if profile == nil || profile.Name == "" || profile.Username == "" || profile.Password == "" {
		return mimo.TerminalError(errors.New("missing registry profile credentials for operator image updates"))
	}

	latestVersion, err := latestOperatorImageVersion(ctx, profile.Name, profile.Username, string(profile.Password))
	if err != nil {
		return err
	}

	currentVersion, hasCurrentVersion := parseOperatorImageVersion(props.OperatorVersion)
	if props.OperatorVersion != "" && !hasCurrentVersion {
		th.SetResultMessage(fmt.Sprintf("skipping operator image auto-update: current operatorVersion %q is not managed by this task", props.OperatorVersion))
		return nil
	}

	if hasCurrentVersion && currentVersion.compare(latestVersion) >= 0 {
		th.SetResultMessage(fmt.Sprintf("operator image already up to date: %s", props.OperatorVersion))
		return nil
	}

	if maintenanceTaskBusy(props.MaintenanceTask) {
		th.SetResultMessage(fmt.Sprintf("skipping operator image auto-update while maintenanceTask=%q", props.MaintenanceTask))
		return nil
	}

	_, err = th.PatchOpenShiftClusterDocument(ctx, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.OperatorVersion = latestVersion.raw
		oscd.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
		return nil
	})
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed to patch cluster doc with operator image update: %w", err))
	}

	if hasCurrentVersion {
		th.SetResultMessage(fmt.Sprintf("scheduled operator image update from %s to %s", currentVersion.raw, latestVersion.raw))
	} else {
		th.SetResultMessage(fmt.Sprintf("scheduled operator image update to %s", latestVersion.raw))
	}
	return nil
}

func latestOperatorImageVersion(ctx context.Context, registryHost, username, password string) (operatorImageVersion, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	tags, err := listACRRepositoryTagsFn(ctx, client, registryHost, operatorImageRepository, username, password)
	if err != nil {
		return operatorImageVersion{}, err
	}

	latest, found := latestOperatorImageVersionFromTags(tags)
	if !found {
		return operatorImageVersion{}, mimo.TerminalError(fmt.Errorf("no valid operator image versions found in %s/%s", registryHost, operatorImageRepository))
	}

	return latest, nil
}

func latestOperatorImageVersionFromTags(tags []string) (operatorImageVersion, bool) {
	var out operatorImageVersion
	found := false
	for _, tag := range tags {
		parsed, ok := parseOperatorImageVersion(tag)
		if !ok {
			continue
		}
		if !found || parsed.compare(out) > 0 {
			out = parsed
			found = true
		}
	}

	return out, found
}

func parseOperatorImageVersion(v string) (operatorImageVersion, bool) {
	match := operatorImageVersionRegex.FindStringSubmatch(strings.TrimSpace(v))
	if len(match) != 3 {
		return operatorImageVersion{}, false
	}

	date, err := strconv.Atoi(match[1])
	if err != nil {
		return operatorImageVersion{}, false
	}

	build, err := strconv.Atoi(match[2])
	if err != nil {
		return operatorImageVersion{}, false
	}

	return operatorImageVersion{
		raw:   v,
		date:  date,
		build: build,
	}, true
}

func (v operatorImageVersion) compare(other operatorImageVersion) int {
	if v.date != other.date {
		return cmpInt(v.date, other.date)
	}
	return cmpInt(v.build, other.build)
}

func cmpInt(x, y int) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	default:
		return 0
	}
}

func maintenanceTaskBusy(task api.MaintenanceTask) bool {
	return !slices.Contains(
		[]api.MaintenanceTask{
			"",
			api.MaintenanceTaskNone,
			api.MaintenanceTaskPending,
		},
		task,
	)
}

func listACRRepositoryTags(ctx context.Context, client *http.Client, registryHost, repository, username, password string) ([]string, error) {
	nextURL := (&url.URL{
		Scheme: "https",
		Host:   registryHost,
		Path:   fmt.Sprintf("/v2/%s/tags/list", repository),
		RawQuery: url.Values{
			"n": []string{strconv.Itoa(operatorTagsPageSize)},
		}.Encode(),
	}).String()

	allTags := []string{}
	for nextURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, mimo.TerminalError(fmt.Errorf("failed creating ACR tags request: %w", err))
		}
		req.SetBasicAuth(username, password)

		resp, err := client.Do(req)
		if err != nil {
			return nil, mimo.TransientError(fmt.Errorf("failed querying ACR tags from %s: %w", registryHost, err))
		}

		err = func() error {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				return mimo.TerminalError(fmt.Errorf("registry credentials rejected by %s (status %d)", registryHost, resp.StatusCode))
			}
			if resp.StatusCode >= http.StatusInternalServerError {
				return mimo.TransientError(fmt.Errorf("registry %s returned retryable status %d", registryHost, resp.StatusCode))
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
				return mimo.TerminalError(fmt.Errorf("registry %s returned status %d: %s", registryHost, resp.StatusCode, strings.TrimSpace(string(body))))
			}

			var body acrTagsListResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				return mimo.TerminalError(fmt.Errorf("failed decoding ACR tags response: %w", err))
			}
			allTags = append(allTags, body.Tags...)

			next, err := parseLinkHeaderNextURL(resp.Request.URL, resp.Header.Get("Link"))
			if err != nil {
				return mimo.TerminalError(fmt.Errorf("failed parsing ACR pagination header: %w", err))
			}
			nextURL = next
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return allTags, nil
}

func parseLinkHeaderNextURL(currentURL *url.URL, linkHeader string) (string, error) {
	if strings.TrimSpace(linkHeader) == "" {
		return "", nil
	}

	parts := strings.Split(linkHeader, ";")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid Link header format: %q", linkHeader)
	}

	if !strings.Contains(parts[1], `rel="next"`) {
		return "", nil
	}

	ref := strings.TrimSpace(parts[0])
	if len(ref) < 2 || ref[0] != '<' || ref[len(ref)-1] != '>' {
		return "", fmt.Errorf("invalid Link header URL segment: %q", parts[0])
	}

	nextRef := ref[1 : len(ref)-1]
	nextURL, err := currentURL.Parse(nextRef)
	if err != nil {
		return "", err
	}

	return nextURL.String(), nil
}

