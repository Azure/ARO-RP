package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/env"
)

type resource string

const (
	ARO               resource = "aro"
	Fluentbit         resource = "fluentbit"
	IfReload          resource = "ifreload"
	MDM               resource = "mdm"
	MDSD              resource = "mdsd"
	OCPRelease        resource = "ocp-release"
	OCPReleaseNightly resource = "ocp-release-nightly"
	OCPARTDev         resource = "ocp-v4.0-art-dev"
	RouteFix          resource = "routefix"
)

type Interface interface {
	GetVersion(resource) string
	ACRName() string
}

type versions struct {
	env env.Lite
	acr string
}

// New is called in RP context
func New(_env env.Lite) (Interface, error) {
	if _env.Type() == env.Dev {
		return &versions{
			env: _env,
			acr: "arointsvc",
		}, nil
	}

	keys := []string{
		"ACR_RESOURCE_ID",
	}

	for _, key := range keys {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	acr, err := azure.ParseResourceID(os.Getenv("ACR_RESOURCE_ID"))
	if err != nil {
		return nil, err
	}

	return &versions{
		env: _env,
		acr: acr.ResourceName,
	}, nil
}

// NewWithACR is called in operator context
func NewWithACR(env env.Lite, acr string) Interface {
	return &versions{
		env: env,
		acr: acr,
	}
}

func (v *versions) GetVersion(r resource) string {
	switch r {
	case ARO:
		if v.env.Type() == env.Dev {
			override := os.Getenv("ARO_IMAGE")
			if override != "" {
				return override
			}
		}
		return v.acr + ".azurecr.io/aro:" + GitCommit
	case Fluentbit:
		return v.acr + ".azurecr.io/fluentbit:1.3.9-1"
	case IfReload:
		return v.acr + ".azurecr.io/ifreload:109810fe"
	case MDM:
		return v.acr + ".azurecr.io/genevamdm:master_41"
	case MDSD:
		return v.acr + ".azurecr.io/genevamdsd:master_295"
	case OCPRelease:
		return v.acr + ".azurecr.io/openshift-release-dev/ocp-release"
	case OCPReleaseNightly:
		return v.acr + ".azurecr.io/openshift-release-dev/ocp-release-nightly"
	case OCPARTDev:
		return v.acr + ".azurecr.io/openshift-release-dev/ocp-v4.0-art-dev"
	case RouteFix:
		return v.acr + ".azurecr.io/routefix:c5c4a5db"
	}

	panic("unimplemented resource " + r)
}

func (v *versions) ACRName() string {
	return v.acr
}
