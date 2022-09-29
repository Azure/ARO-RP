package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/go-test/deep"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestRegions(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset()

	_, log := testlog.New()

	rf := &realFetcher{
		configcli: configcli,
		log:       log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.Regions(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	fileTxt, err := json.Marshal(info)

	f, err := os.Create("data.json")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.Write(fileTxt)

	if err2 != nil {
		log.Fatal(err2)
	}

	txt, err := regionsJsonBytes()
	if err != nil {
		t.Error(err)
		return
	}

	var expected RegionInfo
	err = json.Unmarshal(txt, &expected)
	if err != nil {
		t.Error(err)
	}

	sort.SliceStable(info.Regions, func(i, j int) bool { return info.Regions[i].Name < info.Regions[j].Name })
	sort.SliceStable(expected.Regions, func(i, j int) bool { return expected.Regions[i].Name < expected.Regions[j].Name })

	for _, r := range deep.Equal(expected, info) {
		t.Error(r)
	}
}
