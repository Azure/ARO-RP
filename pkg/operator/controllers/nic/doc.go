package nic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Package nic reconciles failed Network Interface Cards (NICs) in the cluster.
//
// This controller implements a hybrid reconciliation strategy:
// 1. Event-Driven: Watches Machine/MachineSet resources and reconciles their NICs on changes
// 2. Periodic: Scans all NICs in the resource group every hour as a safety net
//
// The controller detects NICs in "Failed" or "Canceled" provisioning states and attempts
// to recover them by triggering Azure re-provisioning.
