package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// The controllers in this package aim to ensure that all required registries are allowed so that cluster remains stable
//
// * Given that customer updates the image.config to add `RequiredRegistries` or `BlockedRegistries`
//
// * ImageConfig controller ensures required registries are allowed / not blocked.
//
// In a little more detail, the controller works as follows:
//

// func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) { watches the ARO object for changes and
// reconciles image.config.openshift.io/cluster object.
// - If blockedRegistries is not nil, makes sure required registries are not added
// - If AllowedRegistries is not nil, makes sure required registries are added
// - Fails fast if both are not nil, unsupported

// func getCloudAwareRegistries(instance *arov1alpha1.Cluster) ([]string, error) { Switch case to ensure the correct registries are added
// depending on the cloud environment (Gov or Public cloud)

// func filterRegistriesInPlace(input []string, keep func(string) bool) []string { Helper function that filters registries to make sure they are added
// in consistent order

// func removeDuplicateRegistries(strSlice []string) []string { Helper function that makes sure only unique registries are added in case controller runs more than once
