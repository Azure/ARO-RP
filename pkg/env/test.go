package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Test struct {
	*prod

	TestSubscriptionID string
	TestLocation       string
	TestResourceGroup  string
	TestDomain         string
}

func (t *Test) Type() Type {
	return Prod
}

func (t *Test) Domain() string {
	return t.TestDomain
}

func (t *Test) Location() string {
	return t.TestLocation
}

func (t *Test) ManagedDomain(clusterDomain string) (string, error) {
	if t.prod == nil {
		t.prod = &prod{}
	}
	t.prod.domain = t.TestDomain
	return t.prod.ManagedDomain(clusterDomain)
}

func (t *Test) ResourceGroup() string {
	return t.TestResourceGroup
}

func (t *Test) SubscriptionID() string {
	return t.TestSubscriptionID
}

func (t *Test) ACRResourceID() string {
	return "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
}

func (t *Test) ACRName() string {
	return "arointsvc"
}
