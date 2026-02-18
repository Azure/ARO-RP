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

	ocb.gatherOperationMetrics(log, operationType, provisioningState, backendErr, dimensions)
	ocb.gatherCorrelationID(log, doc, dimensions)
	ocb.gatherMiscMetrics(log, doc, dimensions)
	ocb.gatherAuthMetrics(log, doc, dimensions)
	ocb.gatherNetworkMetrics(log, doc, dimensions)
	ocb.gatherNodeMetrics(log, doc, dimensions)

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
		var cloudErrorCode string
		if err.CloudErrorBody != nil {
			cloudErrorCode = err.Code
		}
		resultType = utillog.MapStatusCodeToResultType(err.StatusCode, cloudErrorCode)
	}
	return resultType
}

func (ocb *openShiftClusterBackend) getStringMetricValue(log *logrus.Entry, metricName, value string) string {
	if value != "" {
		return value
	}

	log.Warnf("%s %s", metricFailToCollectErr, metricName)
	return empty
}

func (ocb *openShiftClusterBackend) logMetricDimensions(log *logrus.Entry, operationType api.ProvisioningState, dimensions map[string]string) {
	for metric, value := range dimensions {
		log.Info(fmt.Sprintf("%s.%s: %s = %s", metricPackage, operationType, metric, value))
	}
}

func (ocb *openShiftClusterBackend) gatherCorrelationID(log *logrus.Entry, doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.CorrelationData != nil {
		dimensions[correlationDataIdMetricName] = ocb.getStringMetricValue(log, correlationDataIdMetricName, doc.CorrelationData.CorrelationID)
		dimensions[correlationDataClientRequestIdMetricName] = ocb.getStringMetricValue(log, correlationDataClientRequestIdMetricName, doc.CorrelationData.ClientRequestID)
		dimensions[correlationDataRequestIdMetricName] = ocb.getStringMetricValue(log, correlationDataRequestIdMetricName, doc.CorrelationData.RequestID)
	} else {
		log.Warnf("%s %s", metricFailToCollectErr, correlationDataMetricName)
		dimensions[correlationDataIdMetricName] = empty
		dimensions[correlationDataClientRequestIdMetricName] = empty
		dimensions[correlationDataRequestIdMetricName] = empty
	}
}

func (ocb *openShiftClusterBackend) gatherOperationMetrics(log *logrus.Entry, operationType, provisioningState api.ProvisioningState, backendErr error, dimensions map[string]string) {
	// These are provided internally by endLease, not expected to be ""
	dimensions[operationTypeMetricName] = operationType.String()
	dimensions[provisioningStateMetricName] = provisioningState.String()

	dimensions[resultTypeMetricName] = ocb.getStringMetricValue(log, resultTypeMetricName, string(ocb.getResultType(backendErr)))
}

func (ocb *openShiftClusterBackend) gatherMiscMetrics(log *logrus.Entry, doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	dimensions[subscriptionIdMetricName] = ocb.getStringMetricValue(log, subscriptionIdMetricName, ocb.env.SubscriptionID())
	dimensions[resourceIdMetricName] = ocb.getStringMetricValue(log, resourceIdMetricName, doc.ResourceID)

	dimensions[clusterNameMetricName] = ocb.getStringMetricValue(log, clusterNameMetricName, doc.OpenShiftCluster.Name)
	dimensions[clusterIdMetricName] = ocb.getStringMetricValue(log, clusterIdMetricName, doc.OpenShiftCluster.ID)
	dimensions[locationMetricName] = ocb.getStringMetricValue(log, locationMetricName, doc.OpenShiftCluster.Location)
	dimensions[ocpVersionMetricName] = ocb.getStringMetricValue(log, ocpVersionMetricName, doc.OpenShiftCluster.Properties.ClusterProfile.Version)
	dimensions[rpVersionMetricName] = ocb.getStringMetricValue(log, rpVersionMetricName, doc.OpenShiftCluster.Properties.ProvisionedBy)
	dimensions[resourecGroupMetricName] = ocb.getStringMetricValue(log, resourecGroupMetricName, doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)

	for flag, feature := range doc.OpenShiftCluster.Properties.OperatorFlags {
		flagMetricName := fmt.Sprintf("%s-%s", operatorFlagsMetricName, flag)
		dimensions[flagMetricName] = ocb.getStringMetricValue(log, flagMetricName, feature)
	}

	dimensions[asyncOperationsIdMetricName] = ocb.getStringMetricValue(log, asyncOperationsIdMetricName, doc.AsyncOperationID)

	if doc.OpenShiftCluster.Properties.WorkerProfiles != nil {
		dimensions[workerProfileCountMetricName] = strconv.FormatInt(int64(len(doc.OpenShiftCluster.Properties.WorkerProfiles)), 10)
	} else {
		dimensions[workerProfileCountMetricName] = ocb.getStringMetricValue(log, workerProfileCountMetricName, "")
	}

	if doc.OpenShiftCluster.Tags != nil {
		dimensions[tagsMetricName] = enabled
	} else {
		dimensions[tagsMetricName] = disabled
	}
}

func (ocb *openShiftClusterBackend) gatherNodeMetrics(log *logrus.Entry, doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.OpenShiftCluster.Properties.MasterProfile.DiskEncryptionSetID != "" {
		dimensions[masterProfileEncryptionSetIdMetricName] = enabled
	} else {
		dimensions[masterProfileEncryptionSetIdMetricName] = disabled
	}

	mp := doc.OpenShiftCluster.Properties.MasterProfile
	dimensions[masterProfileVmSizeMetricName] = ocb.getStringMetricValue(log, masterProfileVmSizeMetricName, string(mp.VMSize))

	switch doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost {
	case api.EncryptionAtHostEnabled:
		dimensions[masterEncryptionAtHostMetricName] = string(api.EncryptionAtHostEnabled)
	case api.EncryptionAtHostDisabled:
		dimensions[masterEncryptionAtHostMetricName] = string(api.EncryptionAtHostDisabled)
	default:
		log.Warnf("%s %s", metricFailToCollectErr, masterEncryptionAtHostMetricName)
		dimensions[masterEncryptionAtHostMetricName] = unknown
	}

	if len(doc.OpenShiftCluster.Properties.WorkerProfiles) > 0 {
		wp := doc.OpenShiftCluster.Properties.WorkerProfiles[0]
		dimensions[workerVmDiskSizeMetricName] = strconv.FormatInt(int64(wp.DiskSizeGB), 10)
		dimensions[workerVmSizeMetricName] = ocb.getStringMetricValue(log, workerVmSizeMetricName, string(wp.VMSize))
		dimensions[workerVmDiskSizeMetricName] = strconv.FormatInt(int64(wp.DiskSizeGB), 10)

		switch wp.EncryptionAtHost {
		case api.EncryptionAtHostEnabled:
			dimensions[workerEncryptionAtHostMetricName] = string(api.EncryptionAtHostEnabled)
		case api.EncryptionAtHostDisabled:
			dimensions[workerEncryptionAtHostMetricName] = string(api.EncryptionAtHostDisabled)
		default:
			log.Warnf("%s %s", metricFailToCollectErr, workerEncryptionAtHostMetricName)
			dimensions[workerEncryptionAtHostMetricName] = unknown
		}
	}

	switch doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules {
	case api.FipsValidatedModulesEnabled:
		dimensions[fipsMetricName] = string(api.FipsValidatedModulesEnabled)
	case api.FipsValidatedModulesDisabled:
		dimensions[fipsMetricName] = string(api.FipsValidatedModulesDisabled)
	default:
		log.Warnf("%s %s", metricFailToCollectErr, fipsMetricName)
		dimensions[fipsMetricName] = unknown
	}
}

func (ocb *openShiftClusterBackend) gatherAuthMetrics(log *logrus.Entry, doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	if doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile != nil {
		dimensions[clusterIdentityMetricName] = clusterIdentityManagedIdMetricName
	} else if doc.OpenShiftCluster.Properties.ServicePrincipalProfile != nil {
		dimensions[clusterIdentityMetricName] = clusterIdentityServicePrincipalMetricName
	} else {
		log.Warnf("%s %s", metricFailToCollectErr, clusterIdentityMetricName)
		dimensions[clusterIdentityMetricName] = unknown
	}

	if doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret != "" {
		dimensions[pullSecretMetricName] = enabled
	} else {
		dimensions[pullSecretMetricName] = disabled
	}
}

func (ocb *openShiftClusterBackend) gatherNetworkMetrics(log *logrus.Entry, doc *api.OpenShiftClusterDocument, dimensions map[string]string) {
	for _, p := range doc.OpenShiftCluster.Properties.IngressProfiles {
		switch p.Visibility {
		case api.VisibilityPrivate:
			dimensions[ingressProfileMetricName] = fmt.Sprintf("%s.%s", string(api.VisibilityPrivate), p.Name)
		case api.VisibilityPublic:
			dimensions[ingressProfileMetricName] = fmt.Sprintf("%s.%s", string(api.VisibilityPublic), p.Name)
		default:
			log.Warnf("%s %s", metricFailToCollectErr, ingressProfileMetricName)
			dimensions[ingressProfileMetricName] = unknown
		}
	}

	switch doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType {
	case api.OutboundTypeUserDefinedRouting:
		dimensions[networkProfileOutboundTypeMetricName] = string(api.OutboundTypeUserDefinedRouting)
	case api.OutboundTypeLoadbalancer:
		dimensions[networkProfileOutboundTypeMetricName] = string(api.OutboundTypeLoadbalancer)
	default:
		log.Warnf("%s %s", metricFailToCollectErr, networkProfileManagedOutboundIpsMetricName)
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
	if err != nil {
		dimensions[clusterProfileDomainMetricName] = empty
		log.Warnf("%s %s, due to %s", metricFailToCollectErr, clusterProfileDomainMetricName, err.Error())
	} else {
		if domain != "" {
			dimensions[clusterProfileDomainMetricName] = custom
		} else {
			dimensions[clusterProfileDomainMetricName] = managed
		}
	}

	if doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile != nil && doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		dimensions[networkProfileManagedOutboundIpsMetricName] = strconv.FormatInt(int64(doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count), 10)
	} else {
		log.Warnf("%s %s", metricFailToCollectErr, networkProfileManagedOutboundIpsMetricName)
		dimensions[networkProfileManagedOutboundIpsMetricName] = unknown
	}

	switch doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG {
	case api.PreconfiguredNSGEnabled:
		dimensions[networkProfilePreConfiguredNSGMetricName] = string(api.PreconfiguredNSGEnabled)
	case api.PreconfiguredNSGDisabled:
		dimensions[networkProfilePreConfiguredNSGMetricName] = string(api.PreconfiguredNSGDisabled)
	default:
		log.Warnf("%s %s", metricFailToCollectErr, networkProfilePreConfiguredNSGMetricName)
		dimensions[networkProfilePreConfiguredNSGMetricName] = unknown
	}

	if doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		dimensions[featureProfileGatewayEnabledMetricName] = enabled
	} else {
		dimensions[featureProfileGatewayEnabledMetricName] = disabled
	}
}
