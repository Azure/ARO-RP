package scheduler

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type selectorData struct {
	ResourceID string

	SubscriptionID    string
	SubscriptionState api.SubscriptionState
}
