package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// The controllers in this package aim to ensure that MachineConfig objects
// exist and are correctly configured to lay down dnsmasq configuration on
// cluster VMs.  The data path is:
//
// * RP sets Domain, APIIntIP, IngressIP fields on the ARO Cluster object at
//   admin upgrade time.
//
// * ClusterReconciler controller ensures MachineConfigs exist.
//
// * MachineConfigs cause dnsmasq.service and dnsmasq.conf files to be laid down
//   to ensure that DNS resolution of the appropriate names succeeds locally to
//   each VM.
//
// In a little more detail, the controllers work as follows:
//
// func (*ClusterReconciler) Reconcile watches the ARO object for changes and
// reconciles all the 99-%s-aro-dns machineconfigs.
//
// func (*MachineConfigReconciler) Reconcile watches the 99-%s-aro-dns
// machineconfigs in case someone tries to change them, and reconciles them.
//
// func (*MachineConfigPoolReconciler) Reconcile watches MachineConfigPool
// objects and reconciles all the 99-%s-aro-dns machineconfigs.  This covers
// automatically adding dnsmasq configuration to newly created pools.
//
// Internally, all of the above functions call the same reconciler
// (reconcileMachineConfigs).
