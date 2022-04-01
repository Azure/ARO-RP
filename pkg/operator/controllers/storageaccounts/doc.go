package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// The controller in this package aims to ensure that the image registry
// and cluster storage accounts are correctly configured from an access
// perspective.  The data path is:
//
// * RP sets OperatorFlags fields on the ARO Cluster object at
//   admin upgrade or cluster create.
//
// * StorageAccount controller sets the allowed network ruleset and default action
//   to ensure storage accounts are locked down from a network perspective.
//
// In a little more detail, the controller works as follows:
//
// func (*Reconciler) Reconcile watches nodes or machines for updates
// and reconciles the network rulesets and default action in case a new
// node is added from a new subnet to ensure connectivity to the storage
// accounts will exist with the introduction of any node subnets.
//
// This function utilizes two OperatorFlags to determine if the controller is
// enabled and managed.  It operates under the following assumptions:
//
// * If enabled and managed==true, set the default action to Deny and add all
//   node and RP/gateway subnets to the storage accounts' network rulesets
//
// * If enabled and managed==false, set the default action to Allow and add all
//   node and RP/gateway subnets to the storage accounts' network rulesets.
//   Note that we continue to reconcile the network rulesets even though the Allow
//   no longer requires this to simplify the controller logic
//
// * If disabled or managed is not set, do nothing.
