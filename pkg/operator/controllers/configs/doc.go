package configs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Configs controller allows to create custom config that would be applied
// into the cluster.
// The design is inspired by the workaround controller. The difference is
// the focus on configuration instead of hotfixes.
// It also allows for flexible applicable criteria checks since it is
// not fixed on any single item. All checks are done in the Config itself.
// The configs operator takes the cluster config as a main object.
// Other related objects are added in individual configs.
