package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// resourceskus-smoke is a manual smoke test for the streaming ResourceSKUs
// List implementation introduced in ARO-29068.
//
// It hits the real ARM ResourceSKUs API using the current az-cli session and
// reports peak heap allocation, which is the key metric for verifying that
// the streaming approach eliminates the OOM that motivated the change.
//
// Usage:
//
//	go run ./hack/resourceskus-smoke [-mode stream|buffer] [-location eastus]
//
//	  stream  (default) — ARO-29068: streaming JSON decoder, one SKU at a time
//	  buffer            — old approach: NewListPager buffers each full page
//
// Prerequisites: az login must be current.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	localarmcompute "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func main() {
	log := utillog.GetLogger()
	if err := run(log); err != nil {
		log.Fatal(err)
	}
}

func run(log *logrus.Entry) error {
	location := flag.String("location", "eastus", "ARM location filter")
	mode := flag.String("mode", "stream", "stream (ARO-29068) or buffer (old NewListPager)")
	flag.Parse()

	if *mode != "stream" && *mode != "buffer" {
		return fmt.Errorf("-mode must be 'stream' or 'buffer', got %q", *mode)
	}

	subID, err := azureProfileSubscriptionID()
	if err != nil {
		return fmt.Errorf("could not read active subscription from az-cli profile: %w\nRun: az login && az account set -s <subscription>", err)
	}

	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return fmt.Errorf("NewAzureCLICredential: %w", err)
	}

	filter := fmt.Sprintf("location eq %s", *location)

	log.WithFields(logrus.Fields{
		"subscription": subID,
		"mode":         *mode,
		"filter":       filter,
	}).Info("starting")

	ctx := context.Background()

	var seq iter.Seq2[armcompute.ResourceSKU, error]
	switch *mode {
	case "stream":
		client, err := localarmcompute.NewResourceSKUsClient(subID, cred, nil)
		if err != nil {
			return fmt.Errorf("NewResourceSKUsClient: %w", err)
		}
		seq = client.List(ctx, filter, false)

	case "buffer":
		// Old approach: NewListPager fully buffers each page into page.Value
		// before yielding any SKUs to the caller.
		factory, err := armcompute.NewClientFactory(subID, cred, nil)
		if err != nil {
			return fmt.Errorf("NewClientFactory: %w", err)
		}
		pagerClient := factory.NewResourceSKUsClient()
		seq = func(yield func(armcompute.ResourceSKU, error) bool) {
			pager := pagerClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{
				Filter:                   pointerutils.ToPtr(filter),
				IncludeExtendedLocations: pointerutils.ToPtr("false"),
			})
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					yield(armcompute.ResourceSKU{}, err)
					return
				}
				for _, v := range page.Value {
					if !yield(*v, nil) {
						return
					}
				}
			}
		}
	}

	// GC before starting so the baseline is clean.
	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	received := 0
	var peakHeapAlloc uint64

	for sku, err := range seq {
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		_ = sku
		received++

		// Sample heap every 500 SKUs.
		if received%500 == 0 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			if ms.HeapAlloc > peakHeapAlloc {
				peakHeapAlloc = ms.HeapAlloc
			}
			if received%5000 == 0 {
				log.WithFields(logrus.Fields{
					"received": received,
					"heap":     formatBytes(ms.HeapAlloc),
				}).Info("progress")
			}
		}
	}

	elapsed := time.Since(start)

	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	if peakHeapAlloc == 0 {
		peakHeapAlloc = memAfter.HeapAlloc
	}

	log.WithFields(logrus.Fields{
		"mode":          *mode,
		"skus_received": received,
		"elapsed":       elapsed.Round(time.Millisecond).String(),
		"heap_before":   formatBytes(memBefore.HeapAlloc),
		"heap_peak":     formatBytes(peakHeapAlloc),
		"heap_after_gc": formatBytes(memAfter.HeapAlloc),
		"total_alloc":   formatBytes(memAfter.TotalAlloc),
		"sys":           formatBytes(memAfter.Sys),
	}).Info("done")

	return nil
}

// azureProfileSubscriptionID reads the active subscription ID from the az-cli
// profile ($AZURE_CONFIG_DIR/azureProfile.json, default ~/.azure/azureProfile.json).
func azureProfileSubscriptionID() (string, error) {
	configDir := os.Getenv("AZURE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".azure")
	}

	data, err := os.ReadFile(filepath.Join(configDir, "azureProfile.json"))
	if err != nil {
		return "", err
	}

	// az-cli writes a UTF-8 BOM.
	data = []byte(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))

	var profile struct {
		Subscriptions []struct {
			ID        string `json:"id"`
			IsDefault bool   `json:"isDefault"`
		} `json:"subscriptions"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return "", err
	}

	for _, s := range profile.Subscriptions {
		if s.IsDefault {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("no default subscription in azureProfile.json")
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GiB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MiB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KiB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
