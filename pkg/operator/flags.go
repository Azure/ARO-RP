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

	// DNS Type flag for CustomDNS feature
	// Values: "" (default=dnsmasq), "dnsmasq", "clusterhosted"
	// "clusterhosted" enables CustomDNS (CoreDNS static pod) for 4.21+ clusters
	DNSType                            = "aro.dns.type"

	// DNS Type values
	DNSTypeDnsmasq                     = "dnsmasq"
	DNSTypeClusterHosted               = "clusterhosted"

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
	AutosizedNodesEnabled              = "aro.autosizednodes.enabled"
	MuoEnabled                         = "rh.srep.muo.enabled"
	MuoManaged                         = "rh.srep.muo.managed"
	GuardrailsEnabled                  = "aro.guardrails.enabled"
	GuardrailsDeployManaged            = "aro.guardrails.deploy.managed"
	CloudProviderConfigEnabled         = "aro.cloudproviderconfig.enabled"
	ForceReconciliation                = "aro.forcereconciliation"
	EtcHostsEnabled                    = "aro.etchosts.enabled" // true = enable etchosts controller
	EtcHostsManaged                    = "aro.etchosts.managed" // true = apply etchosts mc | false = remove etchosts mc
	FlagTrue                           = "true"
	FlagFalse                          = "false"

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

		// DNS Type: blank (default) = dnsmasq, "dnsmasq" = force dnsmasq, "clusterhosted" = CustomDNS (4.21+)
		DNSType:                            "",

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
		AutosizedNodesEnabled:              FlagTrue,
		MuoEnabled:                         FlagTrue,
		MuoManaged:                         FlagTrue,
		GuardrailsEnabled:                  FlagTrue,
		GuardrailsDeployManaged:            FlagTrue,
		CloudProviderConfigEnabled:         FlagTrue,
		ForceReconciliation:                FlagFalse,
		EtcHostsEnabled:                    FlagTrue,
		EtcHostsManaged:                    FlagTrue,

		// Guardrails policies switches
		GuardrailsPolicyMachineDenyManaged:           FlagTrue,
		GuardrailsPolicyMachineDenyEnforcement:       GuardrailsPolicyDeny,
		GuardrailsPolicyMachineConfigDenyManaged:     FlagTrue,
		GuardrailsPolicyMachineConfigDenyEnforcement: GuardrailsPolicyDryrun,
		GuardrailsPolicyPrivNamespaceDenyManaged:     FlagTrue,
		GuardrailsPolicyPrivNamespaceDenyEnforcement: GuardrailsPolicyDryrun,
	}
}
