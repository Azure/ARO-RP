package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Package dns contains controllers for managing DNS configuration in ARO clusters.
//
// This package supports two DNS implementations:
//   1. dnsmasq (legacy, for clusters <4.21 and manual override)
//   2. CustomDNS / ClusterHostedDNS (for clusters 4.21+, using CoreDNS static pod)
//
// # Controller Enable/Disable
//
// The DNS controller is enabled via the aro.dns.enabled operator flag (new clusters).
// For backward compatibility with existing clusters, the controller also checks
// the legacy aro.dnsmasq.enabled flag if aro.dns.enabled is not present.
//
// # DNS Type Selection
//
// The DNS type is determined by the aro.dns.type operator flag:
//   - "" (blank/default) = dnsmasq (current default)
//   - "dnsmasq" = force dnsmasq usage
//   - "clusterhosted" = CustomDNS (CoreDNS static pod, 4.21+ only)
//
// Version gating ensures "clusterhosted" automatically falls back to dnsmasq
// on clusters older than 4.21.
//
// # Data Flow for dnsmasq Clusters
//
// * RP sets Domain, APIIntIP, IngressIP, GatewayDomains, and GatewayPrivateEndpointIP
//   fields on the ARO Cluster object at install time and admin upgrade time.
//
// * ClusterReconciler controller checks aro.dns.type flag and cluster version,
//   then reconciles MachineConfigs for dnsmasq-based clusters.
//
// * MachineConfigs cause dnsmasq.service and dnsmasq.conf files to be laid down,
//   ensuring DNS resolution of api/api-int/apps domains succeeds locally on each VM.
//
// # Data Flow for CustomDNS Clusters
//
// * RP sets aro.dns.type = "clusterhosted" for new 4.21+ clusters.
//
// * ClusterReconciler detects "clusterhosted" and reconciles the Infrastructure CR's
//   cloudLoadBalancerConfig (status.platformStatus.azure.cloudLoadBalancerConfig)
//   with the cluster's APIIntIP and IngressIP. It skips dnsmasq MachineConfig creation.
//
// * The Infrastructure CR is also watched for drift: if someone modifies
//   the cloudLoadBalancerConfig, the controller restores the correct IPs.
//
// * installer-aro-wrapper reads aro.dns.type from 99_aro.json and sets
//   platform.azure.userProvisionedDNS: Enabled in install-config.
//
// * Upstream installer generates Infrastructure CR with dnsType: ClusterHosted.
//
// * Machine Config Operator (MCO) reads cloudLoadBalancerConfig from the
//   Infrastructure CR and renders CoreDNS static pod on all nodes.
//
// # Controllers
//
// This package contains three reconcilers:
//
// func (*ClusterReconciler) Reconcile watches the ARO Cluster object,
// ClusterVersion, and Infrastructure CR for changes. For dnsmasq clusters,
// it reconciles all 99-%s-aro-dns MachineConfigs. For CustomDNS clusters,
// it reconciles the Infrastructure CR's cloudLoadBalancerConfig with the
// cluster's LB IPs, providing both initial setup and drift protection.
//
// func (*MachineConfigReconciler) Reconcile watches the 99-%s-aro-dns
// MachineConfigs and reconciles them if changes are detected.
//
// func (*MachineConfigPoolReconciler) Reconcile watches MachineConfigPool
// objects and reconciles all the 99-%s-aro-dns MachineConfigs. This covers
// automatically adding dnsmasq configuration to newly created pools.
//
// Internally, all of the above functions call the same reconciler
// (reconcileMachineConfigs) for dnsmasq clusters.
