package feature

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func IsRegisteredForFeature(sub *api.SubscriptionProperties, feature string) bool {
	for _, f := range sub.RegisteredFeatures {
		if f.Name == feature && f.State == "Registered" {
			return true
		}
	}
	return false
}
