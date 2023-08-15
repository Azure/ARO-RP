package rhcos

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"

	coreosarch "github.com/coreos/stream-metadata-go/arch"
	rhcospkg "github.com/openshift/installer/pkg/rhcos"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
)

var rxRHCOS = regexp.MustCompile(`rhcos-((\d+)\.\d+\.\d{8})\d{4}\-\d+-azure\.x86_64\.vhd`)

// Image returns an image object containing VM image SKU information.
func Image(ctx context.Context) (*azuretypes.Image, error) {
	osImage, err := VHD(ctx, types.ArchitectureAMD64)
	if err != nil {
		return nil, err
	}

	m := rxRHCOS.FindStringSubmatch(osImage)
	if m == nil {
		return nil, fmt.Errorf("couldn't match osImage %q", osImage)
	}

	return &azuretypes.Image{
		Publisher: "azureopenshift",
		Offer:     "aro4",
		SKU:       "aro_" + m[2], // "aro_4x"
		Version:   m[1],          // "4x.yy.2020zzzz"
	}, nil
}

// VHD fetches the URL of the public Azure blob containing the RHCOS image
func VHD(ctx context.Context, arch types.Architecture) (string, error) {
	archName := coreosarch.RpmArch(string(arch))
	st, err := rhcospkg.FetchCoreOSBuild(ctx)
	if err != nil {
		return "", err
	}
	streamArch, err := st.GetArchitecture(archName)
	if err != nil {
		return "", err
	}
	ext := streamArch.RHELCoreOSExtensions
	if ext == nil {
		return "", fmt.Errorf("%s: No azure build found", st.FormatPrefix(archName))
	}
	azd := ext.AzureDisk
	if azd == nil {
		return "", fmt.Errorf("%s: No azure build found", st.FormatPrefix(archName))
	}

	return azd.URL, nil
}
