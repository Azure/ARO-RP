package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func (ocb *openShiftClusterBackend) emitMetrics(log *logrus.Entry, doc *api.OpenShiftClusterDocument, operationType, provisioningState api.ProvisioningState, backendErr error) map[string]string {
	dimensions := map[string]string{}

	ocb.gatherOperationMetrics(operationType, provisioningState, backendErr, dimensions)
	ocb.gatherCorrelationID(doc, dimensions)
	ocb.gatherMiscMetrics(doc, dimensions)
	ocb.gatherAuthMetrics(doc, dimensions)
	ocb.gatherNetworkMetrics(doc, dimensions)
	ocb.gatherNodeMetrics(doc, dimensions)

	ocb.logMetricDimensions(log, operationType, dimensions)
	ocb.m.EmitGauge(ocb.getMetricName(operationType), metricValue, dimensions)

	// dimensions is returned here for testing purposes
	return dimensions
}

func (ocb *openShiftClusterBackend) getMetricName(operationType api.ProvisioningState) string {
	return fmt.Sprintf("%s.%s", metricPackage, operationType)
}

func (ocb *openShiftClusterBackend) getResultType(backendErr error) utillog.ResultType {
	var resultType utillog.ResultType
	err, ok := backendErr.(*api.CloudError)
	if ok {
		resultType = utillog.MapStatusCodeToResultType(err.StatusCode)
	}
	return resultType
}

func (ocb *openShiftClusterBackend) logMetricDimensions(log *logrus.Entry, operationType api.ProvisioningState, dimensions map[string]string) {
	for metric, value := range dimensions {
		log.Info(fmt.Sprintf("%s.%s: %s = %s", metricPackage, operationType, metric, value))
	}
}

func (m *openShiftClusterBackend) gatherCorrelationID(doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.CorrelationData != nil {
		dimensions[correlationDataIdMetricName] = doc.CorrelationData.CorrelationID
		dimensions[correlationDataClientRequestIdMetricName] = doc.CorrelationData.ClientRequestID
		dimensions[correlationDataRequestIdMetricName] = doc.CorrelationData.RequestID
	} else {
		dimensions[correlationDataIdMetricName] = empty
		dimensions[correlationDataClientRequestIdMetricName] = empty
		dimensions[correlationDataRequestIdMetricName] = empty
	}
}

func (ocb *openShiftClusterBackend) gatherOperationMetrics(operationType, provisioningState api.ProvisioningState, backendErr error, dimensions map[string]string) {
	dimensions[operationTypeMetricName] = operationType.String()
	dimensions[provisioningStateMetricName] = provisioningState.String()
	dimensions[resultTypeMetricName] = string(ocb.getResultType(backendErr))
}

func (ocb *openShiftClusterBackend) gatherMiscMetrics(doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	dimensions[subscriptionIdMetricName] = ocb.env.SubscriptionID()
	dimensions[resourceIdMetricName] = doc.ResourceID
	if doc.OpenShiftCluster != nil {
		dimensions[clusterNameMetricName] = doc.OpenShiftCluster.Name
		dimensions[locationMetricName] = doc.OpenShiftCluster.Location
		dimensions[ocpVersionMetricName] = doc.OpenShiftCluster.Properties.ClusterProfile.Version
		dimensions[rpVersionMetricName] = doc.OpenShiftCluster.Properties.ProvisionedBy
		dimensions[resourecGroupMetricName] = doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID

		for flag, feature := range doc.OpenShiftCluster.Properties.OperatorFlags {
			dimensions[fmt.Sprintf("%s-%s", operatorFlagsMetricName, flag)] = feature
		}
	}

	dimensions[asyncOperationsIdMetricName] = doc.AsyncOperationID

	if doc.OpenShiftCluster.Properties.WorkerProfiles != nil {
		dimensions[workerProfileCountMetricName] = strconv.FormatInt(int64(len(doc.OpenShiftCluster.Properties.WorkerProfiles)), 10)
	}

	if doc.OpenShiftCluster.Tags != nil {
		dimensions[tagsMetricName] = enabled
	} else {
		dimensions[tagsMetricName] = disabled
	}
}

func (ocb *openShiftClusterBackend) gatherNodeMetrics(doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.OpenShiftCluster.Properties.MasterProfile.DiskEncryptionSetID != "" {
		dimensions[masterProfileEncryptionSetIdMetricName] = enabled
	} else {
		dimensions[masterProfileEncryptionSetIdMetricName] = disabled
	}

	mp := doc.OpenShiftCluster.Properties.MasterProfile
	dimensions[masterProfileVmSizeMetricName] = string(mp.VMSize)

	if doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled {
		dimensions[masterEncryptionAtHostMetricName] = string(api.EncryptionAtHostEnabled)
	} else if doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostDisabled {
		dimensions[masterEncryptionAtHostMetricName] = string(api.EncryptionAtHostDisabled)
	} else {
		dimensions[masterEncryptionAtHostMetricName] = unknown
	}

	if len(doc.OpenShiftCluster.Properties.WorkerProfiles) > 0 {
		wp := doc.OpenShiftCluster.Properties.WorkerProfiles[0]
		dimensions[workerVmSizeMetricName] = string(wp.VMSize)
		dimensions[workerVmDiskSizeMetricName] = strconv.FormatInt(int64(wp.DiskSizeGB), 10)

		dimensions[workerVmSizeMetricName] = string(wp.VMSize)
		dimensions[workerVmDiskSizeMetricName] = strconv.FormatInt(int64(wp.DiskSizeGB), 10)

		if wp.EncryptionAtHost == api.EncryptionAtHostEnabled {
			dimensions[workerEncryptionAtHostMetricName] = string(api.EncryptionAtHostEnabled)
		} else if wp.EncryptionAtHost == api.EncryptionAtHostDisabled {
			dimensions[workerEncryptionAtHostMetricName] = string(api.EncryptionAtHostDisabled)
		} else {
			dimensions[workerEncryptionAtHostMetricName] = unknown
		}
	}

	if doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules == api.FipsValidatedModulesEnabled {
		dimensions[fipsMetricName] = string(api.FipsValidatedModulesEnabled)
	} else if doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules == api.FipsValidatedModulesDisabled {
		dimensions[fipsMetricName] = string(api.FipsValidatedModulesDisabled)
	} else {
		dimensions[fipsMetricName] = unknown
	}
}

func (ocb *openShiftClusterBackend) gatherAuthMetrics(doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile != nil {
		dimensions[clusterIdentityMetricName] = clusterIdentityManagedIdMetricName
	} else if doc.OpenShiftCluster.Properties.ServicePrincipalProfile != nil {
		dimensions[clusterIdentityMetricName] = clusterIdentityServicePrincipalMetricName
	} else {
		dimensions[clusterIdentityMetricName] = unknown
	}

	if doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		dimensions[pullSecretMetricName] = enabled
	} else {
		dimensions[pullSecretMetricName] = disabled
	}
}

func (ocb *openShiftClusterBackend) gatherNetworkMetrics(doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	for _, p := range doc.OpenShiftCluster.Properties.IngressProfiles {
		if p.Visibility == api.VisibilityPrivate {
			dimensions[ingressProfileMetricName] = fmt.Sprintf("%s.%s", string(api.VisibilityPrivate), p.Name)
		} else if p.Visibility == api.VisibilityPublic {
			dimensions[ingressProfileMetricName] = fmt.Sprintf("%s.%s", string(api.VisibilityPublic), p.Name)
		} else {
			dimensions[ingressProfileMetricName] = unknown
		}
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
		dimensions[networkProfileOutboundTypeMetricName] = string(api.OutboundTypeUserDefinedRouting)
	} else if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeLoadbalancer {
		dimensions[networkProfileOutboundTypeMetricName] = string(api.OutboundTypeLoadbalancer)
	} else {
		dimensions[networkProfileOutboundTypeMetricName] = unknown
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.PodCIDR != podCidrDefaultValue {
		dimensions[podCidrMetricName] = custom
	} else {
		dimensions[podCidrMetricName] = defaultSet
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.ServiceCIDR != serviceCidrDefaultValue {
		dimensions[serviceCidrMetricName] = custom
	} else {
		dimensions[serviceCidrMetricName] = defaultSet
	}

	domain, err := dns.ManagedDomain(ocb.env, doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err == nil {
		if domain != "" {
			dimensions[clusterProfileDomainMetricName] = custom
		} else {
			dimensions[clusterProfileDomainMetricName] = managed
		}
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		dimensions[networkProfileManagedOutboundIpsMetricName] = strconv.FormatInt(int64(doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count), 10)
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGEnabled {
		dimensions[networkProfilePreConfiguredNSGMetricName] = string(api.PreconfiguredNSGEnabled)
	} else if doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGDisabled {
		dimensions[networkProfilePreConfiguredNSGMetricName] = string(api.PreconfiguredNSGDisabled)
	} else {
		dimensions[networkProfilePreConfiguredNSGMetricName] = unknown
	}

	if doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		dimensions[featureProfileGatewayEnabledMetricName] = enabled
	} else {
		dimensions[featureProfileGatewayEnabledMetricName] = disabled
	}
}
