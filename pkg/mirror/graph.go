package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func AddFromGraph(channel string, min Version) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.openshift.com/api/upgrades_info/v1/graph", nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = url.Values{
		"channel": []string{fmt.Sprintf("%s-%d.%d", channel, min[0], min[1])},
	}.Encode()

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var g *struct {
		Nodes []struct {
			Version  string                 `json:"version,omitempty"`
			Payload  string                 `json:"payload,omitempty"`
			Metadata map[string]interface{} `json:"metadata,omitempty"`
		} `json:"nodes,omitempty"`
		Edges [][2]int `json:"edges,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&g)
	if err != nil {
		return nil, err
	}

	releases := make([]string, 0, len(g.Nodes))
	for _, node := range g.Nodes {
		version, err := newVersion(node.Version)
		if err != nil {
			return nil, err
		}

		if version.Lt(min) {
			continue
		}

		releases = append(releases, node.Payload)
	}

	return releases, nil
}
