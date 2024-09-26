package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitMetrics(t *testing.T) {
	controller := gomock.NewController(t)
	emitter := mock_metrics.NewMockEmitter(controller)
	env := mock_env.NewMockInterface(controller)
	env.EXPECT().SubscriptionID().AnyTimes()
	env.EXPECT().Domain().AnyTimes()

	log := logrus.NewEntry(&logrus.Logger{})

	b := &backend{
		baseLog: log,
		env:     env,
		m:       emitter,
	}
	ocb := newOpenShiftClusterBackend(b)

	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "resourceGroup"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID, resourceGroup)

	for _, tt := range []struct {
		name              string
		operationType     api.ProvisioningState
		provisioningState api.ProvisioningState
		doc               *api.OpenShiftClusterDocument
		backendErr        error
		managedDomain     bool
	}{
		{
			name: "Pass default cluster install",
			doc: &api.OpenShiftClusterDocument{
				CorrelationData: &api.CorrelationData{
					CorrelationID:   "id",
					ClientRequestID: "client request id",
					RequestID:       "request id",
				},
				ResourceID: resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Location: "eastus",
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Domain:               "cluster.domain.example",
							PullSecret:           api.SecureString("super secret"),
							FipsValidatedModules: api.FipsValidatedModulesDisabled,
						},
						NetworkProfile: api.NetworkProfile{
							LoadBalancerProfile: &api.LoadBalancerProfile{
								ManagedOutboundIPs: &api.ManagedOutboundIPs{
									Count: 1,
								},
							},
							PodCIDR:          podCidrDefaultValue,
							ServiceCIDR:      serviceCidrDefaultValue,
							PreconfiguredNSG: api.PreconfiguredNSGDisabled,
						},
						OperatorFlags:           api.OperatorFlags{"testFlag": "true"},
						WorkerProfiles:          api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles,
						MasterProfile:           api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.MasterProfile,
						ServicePrincipalProfile: &api.ServicePrincipalProfile{},
						IngressProfiles:         api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.IngressProfiles,
						FeatureProfile: api.FeatureProfile{
							GatewayEnabled: true,
						},
					},
				},
			},
			operationType:     api.ProvisioningStateCreating,
			provisioningState: api.ProvisioningStateSucceeded,
		},
		{
			name: "Pass workload identity cluster install",
			doc: &api.OpenShiftClusterDocument{
				CorrelationData: &api.CorrelationData{
					CorrelationID:   "id",
					ClientRequestID: "client request id",
					RequestID:       "request id",
				},
				ResourceID: resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Location: "eastus",
					Tags:     map[string]string{"tag1": "true"},
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Domain:               "cluster.domain.example",
							PullSecret:           api.SecureString("super secret"),
							FipsValidatedModules: api.FipsValidatedModulesEnabled,
						},
						NetworkProfile: api.NetworkProfile{
							LoadBalancerProfile: &api.LoadBalancerProfile{},
							PodCIDR:             "10.128.0.1/14",
							ServiceCIDR:         "172.30.0.1/16",
							PreconfiguredNSG:    api.PreconfiguredNSGEnabled,
						},
						OperatorFlags: api.OperatorFlags{"testFlag": "true"},
						WorkerProfiles: []api.WorkerProfile{
							{
								DiskEncryptionSetID: "testing/disk/encryptionset",
								EncryptionAtHost:    api.EncryptionAtHostEnabled,
							},
						},
						MasterProfile: api.MasterProfile{
							DiskEncryptionSetID: "testing/disk/encryptionset",
							EncryptionAtHost:    api.EncryptionAtHostEnabled,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
						IngressProfiles: []api.IngressProfile{
							{
								Name:       "PrivateIngressProfile",
								Visibility: api.VisibilityPrivate,
							},
						},
						FeatureProfile: api.FeatureProfile{
							GatewayEnabled: true,
						},
					},
				},
			},
			operationType:     api.ProvisioningStateCreating,
			provisioningState: api.ProvisioningStateSucceeded,
		},
		{
			name: "Pass backend error",
			backendErr: &api.CloudError{
				StatusCode: 200,
			},
			doc: &api.OpenShiftClusterDocument{
				CorrelationData: &api.CorrelationData{
					CorrelationID:   "id",
					ClientRequestID: "client request id",
					RequestID:       "request id",
				},
				ResourceID: resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Location: "eastus",
					Tags:     map[string]string{"tag1": "true"},
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Domain:               "cluster",
							PullSecret:           api.SecureString("super secret"),
							FipsValidatedModules: api.FipsValidatedModulesEnabled,
						},
						NetworkProfile: api.NetworkProfile{
							LoadBalancerProfile: &api.LoadBalancerProfile{
								ManagedOutboundIPs: &api.ManagedOutboundIPs{
									Count: 1,
								},
							},
							PodCIDR:     "10.128.0.1/14",
							ServiceCIDR: "172.30.0.1/16",
						},
						OperatorFlags: api.OperatorFlags{"testFlag": "true"},
						WorkerProfiles: []api.WorkerProfile{
							{
								DiskEncryptionSetID: "testing/disk/encryptionset",
								EncryptionAtHost:    api.EncryptionAtHostDisabled,
							},
						},
						MasterProfile: api.MasterProfile{
							DiskEncryptionSetID: "testing/disk/encryptionset",
							EncryptionAtHost:    api.EncryptionAtHostDisabled,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
						IngressProfiles: []api.IngressProfile{
							{
								Name: "EmptyIngressProfile",
							},
						},
						FeatureProfile: api.FeatureProfile{
							GatewayEnabled: true,
						},
					},
				},
			},
			operationType:     api.ProvisioningStateCreating,
			provisioningState: api.ProvisioningStateSucceeded,
			managedDomain:     true,
		},
		{
			name: "Pass UDR Cluster",
			backendErr: &api.CloudError{
				StatusCode: 200,
			},
			doc: &api.OpenShiftClusterDocument{
				CorrelationData: &api.CorrelationData{
					CorrelationID:   "id",
					ClientRequestID: "client request id",
					RequestID:       "request id",
				},
				ResourceID: resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Location: "eastus",
					Tags:     map[string]string{"tag1": "true"},
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							Domain:               "cluster",
							PullSecret:           api.SecureString("super secret"),
							FipsValidatedModules: api.FipsValidatedModulesEnabled,
						},
						NetworkProfile: api.NetworkProfile{
							PodCIDR:     "10.128.0.1/14",
							ServiceCIDR: "172.30.0.1/16",
						},
						OperatorFlags: api.OperatorFlags{"testFlag": "true"},
						WorkerProfiles: []api.WorkerProfile{
							{
								DiskEncryptionSetID: "testing/disk/encryptionset",
								EncryptionAtHost:    api.EncryptionAtHostDisabled,
							},
						},
						MasterProfile: api.MasterProfile{
							DiskEncryptionSetID: "testing/disk/encryptionset",
							EncryptionAtHost:    api.EncryptionAtHostDisabled,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
						IngressProfiles: []api.IngressProfile{
							{
								Name: "EmptyIngressProfile",
							},
						},
						FeatureProfile: api.FeatureProfile{
							GatewayEnabled: true,
						},
					},
				},
			},
			operationType:     api.ProvisioningStateCreating,
			provisioningState: api.ProvisioningStateSucceeded,
			managedDomain:     true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.managedDomain {
				t.Setenv("DOMAIN_NAME", "aro-managed.example")
			}

			dimensions := map[string]string{}
			ocb.gatherOperationMetrics(log, tt.operationType, tt.provisioningState, tt.backendErr, dimensions)
			ocb.gatherCorrelationID(log, tt.doc, dimensions)
			ocb.gatherMiscMetrics(log, tt.doc, dimensions)
			ocb.gatherAuthMetrics(log, tt.doc, dimensions)
			ocb.gatherNetworkMetrics(log, tt.doc, dimensions)
			ocb.gatherNodeMetrics(log, tt.doc, dimensions)

			emitter.EXPECT().EmitGauge(ocb.getMetricName(tt.operationType), metricValue, dimensions).MaxTimes(1)

			d := ocb.emitMetrics(log, tt.doc, tt.operationType, tt.provisioningState, tt.backendErr)

			ok := reflect.DeepEqual(dimensions, d)
			if !ok {
				t.Errorf("%s != %s", dimensions, d)
			}
		})
	}
}
