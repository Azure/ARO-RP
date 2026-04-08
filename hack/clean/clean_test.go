package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

func TestShouldDeleteRespectsPersistTagValue(t *testing.T) {
	t.Parallel()

	s := settings{
		ttl:        time.Hour,
		createdTag: defaultCreatedAtTag,
	}
	log := logrus.NewEntry(logrus.New())
	groupName := "test-rg"
	oldCreatedAt := time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano)

	resourceGroup := mgmtfeatures.ResourceGroup{
		Name: &groupName,
		Tags: map[string]*string{
			"PERSIST":           stringPtr("false"),
			defaultCreatedAtTag: &oldCreatedAt,
		},
	}

	if !s.shouldDelete(resourceGroup, log) {
		t.Fatalf("expected group with persist=false to be deletable")
	}
}

func TestShouldDeleteSkipsOnPersistTrueCaseInsensitive(t *testing.T) {
	t.Parallel()

	s := settings{
		ttl:        time.Hour,
		createdTag: defaultCreatedAtTag,
	}
	log := logrus.NewEntry(logrus.New())
	groupName := "test-rg"
	oldCreatedAt := time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano)

	resourceGroup := mgmtfeatures.ResourceGroup{
		Name: &groupName,
		Tags: map[string]*string{
			"PeRsIsT":           stringPtr("TrUe"),
			defaultCreatedAtTag: &oldCreatedAt,
		},
	}

	if s.shouldDelete(resourceGroup, log) {
		t.Fatalf("expected group with persist=true to be skipped")
	}
}

func TestShouldDeleteReadsCreatedAtTagCaseInsensitive(t *testing.T) {
	t.Parallel()

	s := settings{
		ttl:        time.Hour,
		createdTag: defaultCreatedAtTag,
	}
	log := logrus.NewEntry(logrus.New())
	groupName := "test-rg"
	oldCreatedAt := time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano)

	resourceGroup := mgmtfeatures.ResourceGroup{
		Name: &groupName,
		Tags: map[string]*string{
			"CrEaTeDaT": &oldCreatedAt,
		},
	}

	if !s.shouldDelete(resourceGroup, log) {
		t.Fatalf("expected createdAt lookup to be case-insensitive")
	}
}

func TestShouldDeleteSkipsWhenNameMissing(t *testing.T) {
	t.Parallel()

	s := settings{
		ttl:        time.Hour,
		createdTag: defaultCreatedAtTag,
	}
	log := logrus.NewEntry(logrus.New())
	oldCreatedAt := time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano)

	resourceGroup := mgmtfeatures.ResourceGroup{
		Tags: map[string]*string{
			defaultCreatedAtTag: &oldCreatedAt,
		},
	}

	if s.shouldDelete(resourceGroup, log) {
		t.Fatalf("expected group with missing name to be skipped")
	}
}

func stringPtr(s string) *string {
	return &s
}
