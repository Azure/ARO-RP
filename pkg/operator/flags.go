package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	AlertWebhookEnabled                = "aro.alertwebhook.enabled"
	AzureSubnetsEnabled                = "aro.azuresubnets.enabled"
	AzureSubnetsNsgManaged             = "aro.azuresubnets.nsg.managed"
	AzureSubnetsServiceEndpointManaged = "aro.azuresubnets.serviceendpoint.managed"
	BannerEnabled                      = "aro.banner.enabled"
	CheckerEnabled                     = "aro.checker.enabled"
	CPMSEnabled                        = "aro.cpms.enabled"
	DnsmasqEnabled                     = "aro.dnsmasq.enabled"
	RestartDnsmasqEnabled              = "aro.restartdnsmasq.enabled"
	GenevaLoggingEnabled               = "aro.genevalogging.enabled"
	ImageConfigEnabled                 = "aro.imageconfig.enabled"
	IngressEnabled                     = "aro.ingress.enabled"
	MachineEnabled                     = "aro.machine.enabled"
	MachineSetEnabled                  = "aro.machineset.enabled"
	MachineHealthCheckEnabled          = "aro.machinehealthcheck.enabled"
	MachineHealthCheckManaged          = "aro.machinehealthcheck.managed"
	MonitoringEnabled                  = "aro.monitoring.enabled"
	NodeDrainerEnabled                 = "aro.nodedrainer.enabled"
	PullSecretEnabled                  = "aro.pullsecret.enabled"
	PullSecretManaged                  = "aro.pullsecret.managed"
	RbacEnabled                        = "aro.rbac.enabled"
	RouteFixEnabled                    = "aro.routefix.enabled"
	StorageAccountsEnabled             = "aro.storageaccounts.enabled"
	WorkaroundEnabled                  = "aro.workaround.enabled"
	CopyFailWorkaroundEnabled          = "aro.workaround.copyfail.enabled"
	DirtyfragWorkaroundEnabled         = "aro.workaround.dirtyfrag.enabled"

	// Dynamic workaround catalog. The operator periodically fetches a JSON
	// catalog (over HTTPS) from CatalogURL and applies any matching
	// MachineConfig workarounds. See docs/dynamic-workaround-catalog.md
	// for the manifest schema and rollout guidance.
	//
	// CatalogEnabled is the primary kill switch and must remain reachable
	// via adminUpdate / MIMO operator-flags. When false, the controller
	// removes every MachineConfig it previously applied (identified by the
	// "aro.openshift.io/dynamic-workaround" label).
	//
	// CatalogSecretURI is the full Key Vault secret URI in the form
	//   https://<vault>.vault.azure.net/secrets/<name>[/<version>]
	// The secret value is the v1alpha1 catalog JSON. Auth uses the operator
	// pod's existing AZURE_* env credentials (set from the
	// azure-cloud-credentials Secret) via NewDefaultAzureCredential.
	DynamicWorkaroundCatalogEnabled      = "aro.dynamicworkaround.catalog.enabled"
	DynamicWorkaroundCatalogSecretURI    = "aro.dynamicworkaround.catalog.secretURI"
	DynamicWorkaroundCatalogPollInterval = "aro.dynamicworkaround.catalog.pollinterval"

	// DynamicWorkaroundPredicates carries the per-cluster opt-in: a JSON
	// object mapping a workaround Name (as declared in the catalog) to a CEL
	// boolean expression. A catalog entry applies on this cluster iff its
	// name appears in this map AND the expression evaluates true against the
	// cluster's facts. The catalog itself does not ship predicates; gating
	// lives entirely on the cluster side so the same catalog can roll out to
	// different cluster cohorts independently.
	//
	// Example value (set via adminUpdate / MIMO operator-flags):
	//   {"ipsec-mtu-fix":"ipsecMode == \"Full\" && region == \"eastus\""}
	//
	// Empty value disables every catalog workaround on this cluster (the
	// safe default). See docs/dynamic-workaround-catalog.md for the variable
	// surface and helper functions available to expressions.
	DynamicWorkaroundPredicates = "aro.dynamicworkaround.predicates"

	AutosizedNodesEnabled      = "aro.autosizednodes.enabled"
	MuoEnabled                 = "rh.srep.muo.enabled"
	MuoManaged                 = "rh.srep.muo.managed"
	GuardrailsEnabled          = "aro.guardrails.enabled"
	GuardrailsDeployManaged    = "aro.guardrails.deploy.managed"
	CloudProviderConfigEnabled = "aro.cloudproviderconfig.enabled"
	ForceReconciliation        = "aro.forcereconciliation"
	EtcHostsEnabled            = "aro.etchosts.enabled" // true = enable etchosts controller
	EtcHostsManaged            = "aro.etchosts.managed" // true = apply etchosts mc | false = remove etchosts mc
	FlagTrue                   = "true"
	FlagFalse                  = "false"

	// Guardrails policies switches
	GuardrailsPolicyMachineDenyManaged           = "aro.guardrails.policies.aro-machines-deny.managed"
	GuardrailsPolicyMachineDenyEnforcement       = "aro.guardrails.policies.aro-machines-deny.enforcement"
	GuardrailsPolicyMachineConfigDenyManaged     = "aro.guardrails.policies.aro-machine-config-deny.managed"
	GuardrailsPolicyMachineConfigDenyEnforcement = "aro.guardrails.policies.aro-machine-config-deny.enforcement"
	GuardrailsPolicyPrivNamespaceDenyManaged     = "aro.guardrails.policies.aro-privileged-namespace-deny.managed"
	GuardrailsPolicyPrivNamespaceDenyEnforcement = "aro.guardrails.policies.aro-privileged-namespace-deny.enforcement"
	GuardrailsPolicyDryrun                       = "dryrun"
	GuardrailsPolicyWarn                         = "warn"
	GuardrailsPolicyDeny                         = "deny"
)

// DefaultOperatorFlags returns flags for new clusters
// and ones that have not been AdminUpdated.
func DefaultOperatorFlags() map[string]string {
	return map[string]string{
		AlertWebhookEnabled:                FlagTrue,
		AzureSubnetsEnabled:                FlagTrue,
		AzureSubnetsNsgManaged:             FlagTrue,
		AzureSubnetsServiceEndpointManaged: FlagTrue,
		BannerEnabled:                      FlagFalse,
		CheckerEnabled:                     FlagTrue,
		CPMSEnabled:                        FlagFalse,
		DnsmasqEnabled:                     FlagTrue,
		RestartDnsmasqEnabled:              FlagFalse,
		GenevaLoggingEnabled:               FlagTrue,
		ImageConfigEnabled:                 FlagTrue,
		IngressEnabled:                     FlagTrue,
		MachineEnabled:                     FlagTrue,
		MachineSetEnabled:                  FlagTrue,
		MachineHealthCheckEnabled:          FlagTrue,
		MachineHealthCheckManaged:          FlagTrue,
		MonitoringEnabled:                  FlagFalse,
		NodeDrainerEnabled:                 FlagTrue,
		PullSecretEnabled:                  FlagTrue,
		PullSecretManaged:                  FlagTrue,
		RbacEnabled:                        FlagTrue,
		RouteFixEnabled:                    FlagTrue,
		StorageAccountsEnabled:             FlagTrue,
		WorkaroundEnabled:                  FlagTrue,
		CopyFailWorkaroundEnabled:          FlagTrue,
		DirtyfragWorkaroundEnabled:         FlagTrue,

		// Dynamic workaround catalog is opt-in. Operators set the Key Vault
		// secret URI and flip Enabled to true via adminUpdate / MIMO when a
		// catalog is published; default off everywhere keeps existing clusters
		// untouched until explicitly enabled.
		DynamicWorkaroundCatalogEnabled:      FlagFalse,
		DynamicWorkaroundCatalogSecretURI:    "",
		DynamicWorkaroundCatalogPollInterval: "5m",
		DynamicWorkaroundPredicates:          "",

		AutosizedNodesEnabled:      FlagTrue,
		MuoEnabled:                 FlagTrue,
		MuoManaged:                 FlagTrue,
		GuardrailsEnabled:          FlagTrue,
		GuardrailsDeployManaged:    FlagTrue,
		CloudProviderConfigEnabled: FlagTrue,
		ForceReconciliation:        FlagFalse,
		EtcHostsEnabled:            FlagTrue,
		EtcHostsManaged:            FlagTrue,

		// Guardrails policies switches
		GuardrailsPolicyMachineDenyManaged:           FlagTrue,
		GuardrailsPolicyMachineDenyEnforcement:       GuardrailsPolicyDeny,
		GuardrailsPolicyMachineConfigDenyManaged:     FlagTrue,
		GuardrailsPolicyMachineConfigDenyEnforcement: GuardrailsPolicyDryrun,
		GuardrailsPolicyPrivNamespaceDenyManaged:     FlagTrue,
		GuardrailsPolicyPrivNamespaceDenyEnforcement: GuardrailsPolicyDryrun,
	}
}
