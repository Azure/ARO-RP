package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	metricPackage                = "backend.openshiftcluster"
	metricValue            int64 = 1
	enabled                      = "Enabled"
	disabled                     = "Disabled"
	custom                       = "Custom"
	defaultSet                   = "Default"
	unknown                      = "unknown"
	empty                        = "empty"
	managed                      = "managed"
	metricFailToCollectErr       = "failed to collect metric:"

	encryptionAtHostMetricName = "encryptionathost"
	diskSizeMetricName         = "disksize"
	vmSizeMetricName           = "vmsize"
	countMetricName            = "count"

	workerProfileMetricName          = "workprofile"
	workerVmSizeMetricName           = workerProfileMetricName + "." + vmSizeMetricName
	workerVmDiskSizeMetricName       = workerProfileMetricName + "." + diskSizeMetricName
	workerEncryptionAtHostMetricName = workerProfileMetricName + "." + encryptionAtHostMetricName
	workerProfileCountMetricName     = workerProfileMetricName + "." + countMetricName

	masterProfileMetricName                = "masterprofile"
	masterEncryptionAtHostMetricName       = masterProfileMetricName + "." + encryptionAtHostMetricName
	masterProfileEncryptionSetIdMetricName = masterProfileMetricName + "." + "diskencryptionsetid"
	masterProfileVmSizeMetricName          = masterProfileMetricName + "." + vmSizeMetricName

	fipsMetricName                            = "fips"
	clusterIdentityMetricName                 = "clusteridentity"
	clusterIdentityManagedIdMetricName        = managed + "id"
	clusterIdentityServicePrincipalMetricName = "serviceprincipal"
	pullSecretMetricName                      = "pullsecret"

	ingressProfileMetricName                   = "ingressprofile"
	networkProfileMetricName                   = "networkprofile"
	networkProfileOutboundTypeMetricName       = networkProfileMetricName + "." + "outboundtype"
	networkProfileManagedOutboundIpsMetricName = networkProfileMetricName + "." + "managedoutboundips"
	networkProfilePreConfiguredNSGMetricName   = networkProfileMetricName + "." + "preconfigurednsg"
	podCidrMetricName                          = networkProfileMetricName + "." + "podcidr"
	serviceCidrMetricName                      = networkProfileMetricName + "." + "servicecidr"
	podCidrDefaultValue                        = "10.128.0.0/14"
	serviceCidrDefaultValue                    = "172.30.0.0/16"

	featureProfileMetricName               = "featureprofile"
	featureProfileGatewayEnabledMetricName = featureProfileMetricName + "." + "gatewayenabled"

	clusterProfileMetricName       = "clusterprofile"
	clusterProfileDomainMetricName = clusterProfileMetricName + "." + "domain"

	tagsMetricName          = "tags"
	operatorFlagsMetricName = "operatorflags"

	asyncOperationsIdMetricName = "async_operationsid"
	openshiftClusterMetricName  = "openshiftcluster"
	rpVersionMetricName         = openshiftClusterMetricName + "." + "rpversion"
	ocpVersionMetricName        = openshiftClusterMetricName + "." + "ocpversion"
	clusterNameMetricName       = openshiftClusterMetricName + "." + "clustername"
	clusterIdMetricName         = openshiftClusterMetricName + "." + "clusterid"
	resourecGroupMetricName     = openshiftClusterMetricName + "." + "resourcegroup"
	locationMetricName          = openshiftClusterMetricName + "." + "location"
	resourceIdMetricName        = "resourceid"
	subscriptionIdMetricName    = "subscriptionid"

	correlationDataMetricName                = "correlationdata"
	correlationDataRequestIdMetricName       = correlationDataMetricName + "." + "requestid"
	correlationDataClientRequestIdMetricName = correlationDataMetricName + "." + "client_requestid"
	correlationDataIdMetricName              = correlationDataMetricName + "." + "correlationid"

	operationTypeMetricName     = "operationtype"
	provisioningStateMetricName = "provisioningstate"
	resultTypeMetricName        = "resulttype"
)
