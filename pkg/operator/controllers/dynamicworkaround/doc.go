package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Package dynamicworkaround implements a controller that periodically pulls
// a JSON workaround catalog from an Azure Key Vault secret and applies any
// MachineConfig entries that have been opted in for this cluster. The goal
// is to let RP ship cluster-side mitigations (especially CVE hotfixes that
// take the shape of a MachineConfig) without releasing a new operator image.
//
// Two-key gating:
//
//  1. The catalog is the "menu" of available workarounds. It is published
//     centrally and shipped via Key Vault. The catalog itself does NOT decide
//     which clusters get which workaround.
//
//  2. The per-cluster opt-in lives in the operator flag
//     `aro.dynamicworkaround.predicates` — a JSON object mapping a catalog
//     workaround Name to a CEL boolean expression. A workaround applies on
//     this cluster iff its Name appears in this map AND the expression
//     evaluates true against the cluster's facts (cluster version, region,
//     ipsec mode, architecture).
//
// This split lets the same catalog roll out to different cluster cohorts
// independently: the cohort is just whichever clusters have the right
// predicates flag set, and the predicate refines further (e.g. "only on the
// 4.16.x clusters in eastus").
//
// Trust model: TLS to Key Vault for the catalog, plus Azure RBAC on who can
// publish a catalog and who can set per-cluster operator flags. Both are
// RP-controlled paths — cluster owners cannot point a cluster at an attacker
// catalog and cannot grant themselves a workaround they were not approved
// for.
//
// Kill switch: operator.DynamicWorkaroundCatalogEnabled. When false, every
// MachineConfig previously applied by this controller (identified by the
// "aro.openshift.io/dynamic-workaround" label) is removed.
