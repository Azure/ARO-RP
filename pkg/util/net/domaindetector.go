package net

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	publicCloudManagedDomainSuffix = ".aroapp.io"
	govCloudManagedDomainSuffix    = ".aroapp.azure.us"
)

// DomainDetector is the interface responsible to detect if a cluster has a managed domain or not.
type DomainDetector interface {
	// ClusterHasManagedDomain uses the clusterDomain to detect if the cluster has a managed domain.
	ClusterHasManagedDomain(clusterDomain string) bool
}

// DomainDetectorPublicCloud detects wehether a cluster of the public cloud has a managed domain.
type DomainDetectorPublicCloud struct{}

func (detector *DomainDetectorPublicCloud) ClusterHasManagedDomain(clusterDomain string) bool {
	return strings.HasSuffix(clusterDomain, publicCloudManagedDomainSuffix)
}

// DomainDetectorGovCloud detects wehether a cluster of the gov cloud has a managed domain.
type DomainDetectorGovCloud struct{}

func (detector *DomainDetectorGovCloud) ClusterHasManagedDomain(clusterDomain string) bool {
	return strings.HasSuffix(clusterDomain, govCloudManagedDomainSuffix)
}

// NewDomainDetector returns a DomainDetector based on the cloudName.
func NewDomainDetector(cloudName string) (DomainDetector, error) {
	aroEnvironment, err := azureclient.EnvironmentFromName(cloudName)
	if err != nil {
		return nil, err
	}

	if aroEnvironment.ActualCloudName == azureclient.PublicCloud.ActualCloudName {
		return &DomainDetectorPublicCloud{}, nil
	}

	if aroEnvironment.ActualCloudName == azureclient.USGovernmentCloud.ActualCloudName {
		return &DomainDetectorGovCloud{}, nil
	}

	return nil, errors.New("invalid cloudName")
}

type FakeDomainDetector struct {
	HasManagedDomain bool
}

func (f *FakeDomainDetector) ClusterHasManagedDomain(clusterDomain string) bool {
	return f.HasManagedDomain
}
