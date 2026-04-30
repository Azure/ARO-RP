package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleSubscriptionDocument() *SubscriptionDocument {
	return &SubscriptionDocument{
		ID: "00000000-0000-0000-0000-000000000000",
		Subscription: &Subscription{
			State: SubscriptionStateRegistered,
			Properties: &SubscriptionProperties{
				TenantID: "11111111-1111-1111-1111-111111111111",
			},
		},
	}
}
