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
	AutosizedNodesEnabled              = "aro.autosizednodes.enabled"
	MuoEnabled                         = "rh.srep.muo.enabled"
	MuoManaged                         = "rh.srep.muo.managed"
	GuardrailsEnabled                  = "aro.guardrails.enabled"
	GuardrailsDeployManaged            = "aro.guardrails.deploy.managed"
	CloudProviderConfigEnabled         = "aro.cloudproviderconfig.enabled"
	ForceReconciliation                = "aro.forcereconciliation"
	FlagTrue                           = "true"
	FlagFalse                          = "false"
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
		DnsmasqEnabled:                     FlagTrue,
		RestartDnsmasqEnabled:              FlagFalse,
		GenevaLoggingEnabled:               FlagTrue,
		ImageConfigEnabled:                 FlagTrue,
		IngressEnabled:                     FlagTrue,
		MachineEnabled:                     FlagTrue,
		MachineSetEnabled:                  FlagTrue,
		MachineHealthCheckEnabled:          FlagTrue,
		MachineHealthCheckManaged:          FlagTrue,
		MonitoringEnabled:                  FlagTrue,
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
		GuardrailsEnabled:                  FlagFalse,
		GuardrailsDeployManaged:            FlagFalse,
		CloudProviderConfigEnabled:         FlagTrue,
		ForceReconciliation:                FlagFalse,
	}
}
