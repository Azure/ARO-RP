package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Node struct {
	Version  string                 `json:"version,omitempty"`
	Payload  string                 `json:"payload,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AddFromGraph adds all nodes whose version is of the form x.y.z (no suffix)
// and >= min
func AddFromGraph(min version.Version) ([]Node, error) {
	req, err := http.NewRequest(http.MethodGet, "https://amd64.ocp.releases.ci.openshift.org/graph", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	if mediaType != "application/vnd.redhat.cincinnati.graph+json" {
		return nil, fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var g *struct {
		Nodes []Node   `json:"nodes,omitempty"`
		Edges [][2]int `json:"edges,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&g)
	if err != nil {
		return nil, err
	}

	releases := make([]Node, 0, len(g.Nodes))
	for _, node := range g.Nodes {
		vsn, err := version.ParseVersion(node.Version)
		if err != nil {
			return nil, err
		}

		// if incoming version < min - skip
		if vsn.Lt(min) || vsn.Suffix != "" {
			continue
		}

		node.Payload = strings.Replace(node.Payload, "registry.ci.openshift.org/ocp/release", "quay.io/openshift-release-dev/ocp-release", 1)

		releases = append(releases, node)
	}

	return releases, nil
}

// VersionInfo fetches the Node containing the version payload
func VersionInfo(ver version.Version) (Node, error) {
	nodes, err := AddFromGraph(ver)
	if err != nil {
		return Node{}, err
	}

	for _, node := range nodes {
		if strings.EqualFold(node.Version, ver.String()) {
			return node, nil
		}
	}

	return Node{}, fmt.Errorf("version '%s' not found", ver.String())
}
