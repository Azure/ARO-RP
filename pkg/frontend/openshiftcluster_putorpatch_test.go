package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	v20220401 "github.com/Azure/ARO-RP/pkg/api/v20220401"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_frontend "github.com/Azure/ARO-RP/pkg/util/mocks/frontend"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

const (
	mockGuid          = "00000000-0000-0000-0000-000000000000"
	mockMiResourceId  = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/not-a-real-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/not-a-real-mi"
	mockMiResourceId2 = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/not-a-real-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/not-a-real-mi-2"
)

var (
	defaultVersion                    = version.DefaultInstallStream.Version.String()
	defaultMinorVersion               = version.DefaultInstallStream.Version.MinorVersion()
	unexpectedWorkloadIdentitiesError = fmt.Sprintf(`400: PlatformWorkloadIdentityMismatch: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[aro-operator cloud-controller-manager cloud-network-config disk-csi-driver file-csi-driver image-registry ingress machine-api]'`, defaultMinorVersion)
)

type dummyOpenShiftClusterValidator struct{}

func (*dummyOpenShiftClusterValidator) Static(interface{}, *api.OpenShiftCluster, string, string, bool, api.ArchitectureVersion, string) error {
	return nil
}

func TestPutOrPatchOpenShiftClusterAdminAPI(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"admin": {
			OpenShiftClusterConverter:            api.APIs["admin"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator:      api.APIs["admin"].OpenShiftClusterStaticValidator,
			OpenShiftClusterCredentialsConverter: api.APIs["admin"].OpenShiftClusterCredentialsConverter,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name                   string
		request                func(*admin.OpenShiftCluster)
		isPatch                bool
		fixture                func(*testdatabase.Fixture)
		wantStatusCode         int
		wantEnriched           []string
		wantDocuments          func(*testdatabase.Checker)
		wantResponse           func() *admin.OpenShiftCluster
		wantAsync              bool
		wantError              string
		wantSystemDataEnriched bool
	}

	for _, tt := range []*test{
		{
			name: "patch with empty request",
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							OperatorFlags:     api.OperatorFlags{"testFlag": "true"},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: admin.MaintenanceTaskEverything,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			}},
		{
			name: "patch with flags merges the flags together",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
				oc.Properties.OperatorFlags = admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true"}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",

						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							ProvisioningState: api.ProvisioningStateSucceeded,
							OperatorFlags:     api.OperatorFlags{"testFlag": "true", "overwrittenFlag": "false"},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       api.MaintenanceTaskOperator,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true", "testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						MaintenanceTask:       admin.MaintenanceTaskOperator,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true", "testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			}},
		{
			name: "patch an existing cluster with no flags in db will use defaults",
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    operator.DefaultOperatorFlags(),
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						MaintenanceTask:       admin.MaintenanceTaskEverything,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						OperatorFlags:    admin.OperatorFlags(operator.DefaultOperatorFlags()),
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			}},
		{
			name: "patch with operator update request",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskOperator,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    operator.DefaultOperatorFlags(),
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: admin.MaintenanceTaskOperator,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags(operator.DefaultOperatorFlags()),
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			},
		},
		{
			name: "patch with OperatorFlagsMergeStrategy=reset will reset flags to defaults and merge in request flags",
			request: func(oc *admin.OpenShiftCluster) {
				oc.OperatorFlagsMergeStrategy = admin.OperatorFlagsMergeStrategyReset
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
				oc.Properties.OperatorFlags = admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true"}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",

						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							ProvisioningState: api.ProvisioningStateSucceeded,
							OperatorFlags:     api.OperatorFlags{"testFlag": "true", "overwrittenFlag": "false"},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				expectedFlags := operator.DefaultOperatorFlags()
				expectedFlags["exploding-flag"] = "true"
				expectedFlags["overwrittenFlag"] = "true"

				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       api.MaintenanceTaskOperator,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags(expectedFlags),
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				expectedFlags := operator.DefaultOperatorFlags()
				expectedFlags["exploding-flag"] = "true"
				expectedFlags["overwrittenFlag"] = "true"

				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						MaintenanceTask:       admin.MaintenanceTaskOperator,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags(expectedFlags),
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			},
		},
		{
			name: "patch with operator update request -- existing maintenance task",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:   api.MaintenanceTaskEverything,
							OperatorFlags:     api.OperatorFlags{"testFlag": "true"},
							MaintenanceState:  api.MaintenanceStateNone,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskOperator,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: admin.MaintenanceTaskOperator,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			},
		},
		{
			name: "patch a cluster with registry profile should fail",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.RegistryProfiles = []admin.RegistryProfile{
					{
						Name:     "TestUser",
						Username: "TestUserName",
					},
				}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					},
				})
			},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					}})
			},
			wantSystemDataEnriched: false,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:              false,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              `400: PropertyChangeNotAllowed: properties.registryProfiles: Changing property 'properties.registryProfiles' is not allowed.`,
			wantResponse:           func() *admin.OpenShiftCluster { return nil },
		},
		{
			name: "patch an empty maintenance state cluster with maintenance pending request",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskPending
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       "",
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							MaintenanceState: api.MaintenanceStatePending,
							OperatorFlags:    operator.DefaultOperatorFlags(),
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateSucceeded,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: "",
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MaintenanceState: admin.MaintenanceStatePending,
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags: admin.OperatorFlags(operator.DefaultOperatorFlags()),
					},
				}
			},
		},
		{
			name: "patch a none maintenance state cluster with maintenance pending request",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskPending
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       "",
							MaintenanceState:      api.MaintenanceStateNone,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							MaintenanceState: api.MaintenanceStatePending,
							OperatorFlags:    operator.DefaultOperatorFlags(),
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateSucceeded,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: "",
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MaintenanceState: admin.MaintenanceStatePending,
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags: admin.OperatorFlags(operator.DefaultOperatorFlags()),
					},
				}
			},
		},
		{
			name: "patch a maintenance state pending cluster with planned maintenance",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskEverything
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceState:      api.MaintenanceStatePending,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    operator.DefaultOperatorFlags(),
							MaintenanceState: api.MaintenanceStatePlanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: admin.MaintenanceTaskEverything,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags(operator.DefaultOperatorFlags()),
						MaintenanceState: admin.MaintenanceStatePlanned,
					},
				}
			},
		},
		{
			name: "patch a planned maintenance ongoing cluster with maintenance none request",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskNone
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       "",
							MaintenanceState:      api.MaintenanceStatePlanned,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							MaintenanceState: api.MaintenanceStateNone,
							OperatorFlags:    operator.DefaultOperatorFlags(),
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateSucceeded,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: "",
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MaintenanceState: admin.MaintenanceStateNone,
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags: admin.OperatorFlags(operator.DefaultOperatorFlags()),
					},
				}
			},
		},
		{
			name: "patch an unplanned maintenance ongoing cluster with maintenance none request",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskNone
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceTask:       "",
							MaintenanceState:      api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							MaintenanceState: api.MaintenanceStateNone,
							OperatorFlags:    operator.DefaultOperatorFlags(),
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateSucceeded,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: "",
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MaintenanceState: admin.MaintenanceStateNone,
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags: admin.OperatorFlags(operator.DefaultOperatorFlags()),
					},
				}
			},
		},
		{
			name: "patch a none maintenance state cluster with maintenance unplanned request",
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							OperatorFlags:     api.OperatorFlags{"testFlag": "true"},
							MaintenanceState:  api.MaintenanceStateNone,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						MaintenanceTask: admin.MaintenanceTaskEverything,
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
					},
				}
			},
		},
		{
			name: "patch a failed planned maintenance cluster with customer action needed",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskCustomerActionNeeded
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							OperatorFlags:         api.OperatorFlags{"testFlag": "true"},
							LastAdminUpdateError:  "error",
							MaintenanceState:      api.MaintenanceStatePlanned,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateSucceeded,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:        api.OperatorFlags{"testFlag": "true"},
							MaintenanceState:     api.MaintenanceStateCustomerActionNeeded,
							LastAdminUpdateError: "error",
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateSucceeded,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:        admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState:     admin.MaintenanceStateCustomerActionNeeded,
						LastAdminUpdateError: "error",
					},
				}
			},
		},
		{
			name: "patch a failed planned maintenance cluster with customer action needed",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskCustomerActionNeeded
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							OperatorFlags:           api.OperatorFlags{"testFlag": "true"},
							LastAdminUpdateError:    "error",
							MaintenanceState:        api.MaintenanceStatePlanned,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:        api.OperatorFlags{"testFlag": "true"},
							MaintenanceState:     api.MaintenanceStateCustomerActionNeeded,
							LastAdminUpdateError: "error",
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:       admin.ProvisioningStateSucceeded,
						FailedProvisioningState: admin.ProvisioningStateUpdating,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:        admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState:     admin.MaintenanceStateCustomerActionNeeded,
						LastAdminUpdateError: "error",
					},
				}
			},
		},
		{
			name: "patch a failed unplanned maintenance cluster with customer action needed",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskCustomerActionNeeded
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							OperatorFlags:           api.OperatorFlags{"testFlag": "true"},
							LastAdminUpdateError:    "error",
							MaintenanceState:        api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:        api.OperatorFlags{"testFlag": "true"},
							MaintenanceState:     api.MaintenanceStateCustomerActionNeeded,
							LastAdminUpdateError: "error",
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:       admin.ProvisioningStateSucceeded,
						FailedProvisioningState: admin.ProvisioningStateUpdating,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:        admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState:     admin.MaintenanceStateCustomerActionNeeded,
						LastAdminUpdateError: "error",
					},
				}
			},
		},
		{
			name: "patch a customer action needed cluster with maintenance state none",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskNone
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							OperatorFlags:           api.OperatorFlags{"testFlag": "true"},
							LastAdminUpdateError:    "error",
							MaintenanceState:        api.MaintenanceStateCustomerActionNeeded,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateSucceeded,
						ProvisioningState:        api.ProvisioningStateSucceeded,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							LastAdminUpdateError: "error",
							MaintenanceTask:      "",
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateNone,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:       admin.ProvisioningStateSucceeded,
						FailedProvisioningState: admin.ProvisioningStateUpdating,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:        admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState:     admin.MaintenanceStateNone,
						LastAdminUpdateError: "error",
					},
				}
			},
		},
		{
			name: "patch a customer action needed cluster with maintenance state unplanned",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.MaintenanceTask = admin.MaintenanceTaskEverything
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							OperatorFlags:           api.OperatorFlags{"testFlag": "true"},
							LastAdminUpdateError:    "error",
							MaintenanceState:        api.MaintenanceStateCustomerActionNeeded,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateAdminUpdating,
							LastProvisioningState:   api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MaintenanceTask: api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:       admin.ProvisioningStateAdminUpdating,
						LastProvisioningState:   admin.ProvisioningStateSucceeded,
						FailedProvisioningState: admin.ProvisioningStateUpdating,
						ClusterProfile: admin.ClusterProfile{
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
						},
						NetworkProfile: admin.NetworkProfile{
							OutboundType:     admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
						MaintenanceTask:  admin.MaintenanceTaskEverything,
					},
				}
			},
		},
		{
			name: "patch workload identity cluster with empty request - WI-related fields are still present in the cluster doc afterward",
			// Several workload identity-related fields are removed from the cluster doc by the converter's
			// ExternalNoReadOnly. Since the `wantDocuments` and `wantResponse` fields in these unit test cases
			// only reflect what the frontend does before the backend acts on the async operation, these fields,
			// which were included in the fixture, are notably missing in `wantDocuments` and `wantResponse`. In an
			// end-to-end admin PATCH, the fields would be repopulated by designated steps in the admin update.
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Identity: &api.ManagedServiceIdentity{
							Type: api.ManagedServiceIdentityUserAssigned,
							UserAssignedIdentities: map[string]api.UserAssignedIdentity{
								mockMiResourceId: {
									ClientID:    mockGuid,
									PrincipalID: mockGuid,
								},
							},
							IdentityURL: middleware.MockIdentityURL,
							TenantID:    mockGuid,
						},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							ClusterProfile: api.ClusterProfile{
								Version:                       defaultVersion,
								OIDCIssuer:                    (*api.OIDCIssuer)(pointerutils.ToPtr(mockGuid)),
								BoundServiceAccountSigningKey: (*api.SecureString)(pointerutils.ToPtr(mockGuid)),
							},
							OperatorFlags: api.OperatorFlags{"testFlag": "true"},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
								},
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantEnriched:           []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Identity: &api.ManagedServiceIdentity{
							Type: api.ManagedServiceIdentityUserAssigned,
							UserAssignedIdentities: map[string]api.UserAssignedIdentity{
								mockMiResourceId: {},
							},
							IdentityURL: middleware.MockIdentityURL,
						},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:       []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							ClusterProfile: api.ClusterProfile{
								Version:                       defaultVersion,
								FipsValidatedModules:          api.FipsValidatedModulesDisabled,
								OIDCIssuer:                    (*api.OIDCIssuer)(pointerutils.ToPtr(mockGuid)),
								BoundServiceAccountSigningKey: (*api.SecureString)(pointerutils.ToPtr(mockGuid)),
							},
							MaintenanceTask: api.MaintenanceTaskEverything,
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags:    api.OperatorFlags{"testFlag": "true"},
							MaintenanceState: api.MaintenanceStateUnplanned,
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
									},
								},
							},
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Identity: &admin.ManagedServiceIdentity{
						Type: admin.ManagedServiceIdentityUserAssigned,
						UserAssignedIdentities: map[string]admin.UserAssignedIdentity{
							mockMiResourceId: {},
						},
					},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						IngressProfiles:       []admin.IngressProfile{{Name: "default"}},
						WorkerProfiles: []admin.WorkerProfile{
							{
								Name:             "default",
								EncryptionAtHost: admin.EncryptionAtHostDisabled,
							},
						},
						ClusterProfile: admin.ClusterProfile{
							Version:              defaultVersion,
							FipsValidatedModules: admin.FipsValidatedModulesDisabled,
							OIDCIssuer:           (*admin.OIDCIssuer)(pointerutils.ToPtr(mockGuid)),
						},
						MaintenanceTask: admin.MaintenanceTaskEverything,
						NetworkProfile: admin.NetworkProfile{
							SoftwareDefinedNetwork: admin.SoftwareDefinedNetworkOpenShiftSDN,
							OutboundType:           admin.OutboundTypeLoadbalancer,
							PreconfiguredNSG:       admin.PreconfiguredNSGDisabled,
							LoadBalancerProfile: &admin.LoadBalancerProfile{
								ManagedOutboundIPs: &admin.ManagedOutboundIPs{
									Count: 1,
								},
							},
						},
						MasterProfile: admin.MasterProfile{
							EncryptionAtHost: admin.EncryptionAtHostDisabled,
						},
						OperatorFlags:    admin.OperatorFlags{"testFlag": "true"},
						MaintenanceState: admin.MaintenanceStateUnplanned,
						PlatformWorkloadIdentityProfile: &admin.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]admin.PlatformWorkloadIdentity{
								"file-csi-driver": {
									ResourceID: mockMiResourceId,
								},
								"cloud-controller-manager": {
									ResourceID: mockMiResourceId,
								},
								"ingress": {
									ResourceID: mockMiResourceId,
								},
								"image-registry": {
									ResourceID: mockMiResourceId,
								},
								"machine-api": {
									ResourceID: mockMiResourceId,
								},
								"cloud-network-config": {
									ResourceID: mockMiResourceId,
								},
								"aro-operator": {
									ResourceID: mockMiResourceId,
								},
								"disk-csi-driver": {
									ResourceID: mockMiResourceId,
								},
							},
						},
					},
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithAsyncOperations().
				WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, apis, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := &admin.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			resp, b, err := ti.request(method,
				"https://server"+testdatabase.GetResourcePath(mockSubID, "resourceName")+"?api-version=admin",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, oc)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}
			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse())
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
			}
			errs := ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)
			for _, i := range errs {
				t.Error(i)
			}
			errs = ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
			for _, i := range errs {
				t.Error(i)
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

func TestPutOrPatchOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2024-08-12-preview": {
			OpenShiftClusterConverter:            api.APIs["2024-08-12-preview"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator:      &dummyOpenShiftClusterValidator{},
			OpenShiftClusterCredentialsConverter: api.APIs["2024-08-12-preview"].OpenShiftClusterCredentialsConverter,
		},
	}

	mockCurrentTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	type test struct {
		name                    string
		request                 func(*v20240812preview.OpenShiftCluster)
		isPatch                 bool
		fixture                 func(*testdatabase.Fixture)
		ocpVersionsChangeFeed   map[string]*api.OpenShiftVersion
		quotaValidatorError     error
		skuValidatorError       error
		providersValidatorError error
		wantEnriched            []string
		wantSystemDataEnriched  bool
		wantDocuments           func(*testdatabase.Checker)
		wantStatusCode          int
		wantResponse            *v20240812preview.OpenShiftCluster
		wantAsync               bool
		wantError               string
	}

	for _, tt := range []*test{
		{
			name: "create a new cluster",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Properties.ServicePrincipalProfile = &v20240812preview.ServicePrincipalProfile{
					ClientID: mockGuid,
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateCreating,
						ProvisioningState:        api.ProvisioningStateCreating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:         strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					ClientIDKey: mockGuid,
					Bucket:      1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: version.InstallArchitectureVersion,
							ProvisioningState:   api.ProvisioningStateCreating,
							ProvisionedBy:       version.GitCommit,
							CreatedAt:           mockCurrentTime,
							CreatedBy:           version.GitCommit,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientID: mockGuid,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							FeatureProfile: api.FeatureProfile{
								GatewayEnabled: true,
							},
							OperatorFlags: operator.DefaultOperatorFlags(),
						},
					},
				})
			},
			wantEnriched:   []string{},
			wantAsync:      true,
			wantStatusCode: http.StatusCreated,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateCreating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Version:              defaultVersion,
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					ServicePrincipalProfile: &v20240812preview.ServicePrincipalProfile{
						ClientID: mockGuid,
					},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "create a new workload identity cluster",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"disk-csi-driver":          {ResourceID: mockMiResourceId},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateCreating,
						ProvisioningState:        api.ProvisioningStateCreating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:         strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					ClientIDKey: strings.ToLower(mockMiResourceId),
					Bucket:      1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Identity: &api.ManagedServiceIdentity{
							Type: "UserAssigned",
							UserAssignedIdentities: map[string]api.UserAssignedIdentity{
								mockMiResourceId: {},
							},
							IdentityURL: middleware.MockIdentityURL,
							TenantID:    mockGuid,
						},
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: version.InstallArchitectureVersion,
							ProvisioningState:   api.ProvisioningStateCreating,
							ProvisionedBy:       version.GitCommit,
							CreatedAt:           mockCurrentTime,
							CreatedBy:           version.GitCommit,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							FeatureProfile: api.FeatureProfile{
								GatewayEnabled: true,
							},
							OperatorFlags: operator.DefaultOperatorFlags(),
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver":          {ResourceID: mockMiResourceId},
									"cloud-controller-manager": {ResourceID: mockMiResourceId},
									"ingress":                  {ResourceID: mockMiResourceId},
									"image-registry":           {ResourceID: mockMiResourceId},
									"machine-api":              {ResourceID: mockMiResourceId},
									"cloud-network-config":     {ResourceID: mockMiResourceId},
									"aro-operator":             {ResourceID: mockMiResourceId},
									"disk-csi-driver":          {ResourceID: mockMiResourceId},
								},
							},
						},
					},
				})
			},
			wantEnriched:   []string{},
			wantAsync:      true,
			wantStatusCode: http.StatusCreated,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Identity: &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
					TenantID: mockGuid,
				},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateCreating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Version:              defaultVersion,
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
					PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
							"file-csi-driver":          {ResourceID: mockMiResourceId},
							"cloud-controller-manager": {ResourceID: mockMiResourceId},
							"ingress":                  {ResourceID: mockMiResourceId},
							"image-registry":           {ResourceID: mockMiResourceId},
							"machine-api":              {ResourceID: mockMiResourceId},
							"cloud-network-config":     {ResourceID: mockMiResourceId},
							"aro-operator":             {ResourceID: mockMiResourceId},
							"disk-csi-driver":          {ResourceID: mockMiResourceId},
						},
					},
				},
			},
		},
		{
			name: "create a new workload identity cluster - unexpected workload identity provided",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"unexpected-identity":      {ResourceID: mockMiResourceId},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantEnriched:           []string{},
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new workload identity cluster - missing workload identity provided",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantEnriched:           []string{},
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new workload identity cluster - extra workload identity provided",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"disk-csi-driver":          {ResourceID: mockMiResourceId},
						"extra-identity":           {ResourceID: mockMiResourceId},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantEnriched:           []string{},
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new cluster vm not supported",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed: getOCPVersionsChangeFeed(),
			quotaValidatorError:   api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The provided VM SKU %s is not supported.", "something")),
			wantEnriched:          []string{},
			wantStatusCode:        http.StatusBadRequest,
			wantError:             "400: InvalidParameter: : The provided VM SKU something is not supported.",
		},
		{
			name: "create a new cluster quota fails",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed: getOCPVersionsChangeFeed(),
			quotaValidatorError:   api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeQuotaExceeded, "", "Resource quota of vm exceeded. Maximum allowed: 0, Current in use: 0, Additional requested: 1."),
			wantEnriched:          []string{},
			wantStatusCode:        http.StatusBadRequest,
			wantError:             "400: QuotaExceeded: : Resource quota of vm exceeded. Maximum allowed: 0, Current in use: 0, Additional requested: 1.",
		},
		{
			name: "create a new cluster sku unavailable",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed: getOCPVersionsChangeFeed(),
			skuValidatorError:     api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", "Standard_Sku", "somewhere")),
			wantEnriched:          []string{},
			wantStatusCode:        http.StatusBadRequest,
			wantError:             "400: InvalidParameter: : The selected SKU 'Standard_Sku' is unavailable in region 'somewhere'",
		},
		{
			name: "create a new cluster sku restricted",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed: getOCPVersionsChangeFeed(),
			skuValidatorError:     api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The selected SKU '%v' is restricted in region '%v' for selected subscription", "Standard_Sku", "somewhere")),
			wantEnriched:          []string{},
			wantStatusCode:        http.StatusBadRequest,
			wantError:             "400: InvalidParameter: : The selected SKU 'Standard_Sku' is restricted in region 'somewhere' for selected subscription",
		},

		{
			name: "create a new cluster Microsoft.Authorization provider not registered",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:   getOCPVersionsChangeFeed(),
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Authorization")),
			wantEnriched:            []string{},
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Authorization' is not registered.",
		},
		{
			name: "create a new cluster Microsoft.Compute provider not registered",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:   getOCPVersionsChangeFeed(),
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Compute")),
			wantEnriched:            []string{},
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "create a new cluster Microsoft.Network provider not registered",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:   getOCPVersionsChangeFeed(),
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Network")),
			wantEnriched:            []string{},
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Network' is not registered.",
		},
		{
			name: "create a new cluster Microsoft.Storage provider not registered",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			ocpVersionsChangeFeed:   getOCPVersionsChangeFeed(),
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Storage")),
			wantEnriched:            []string{},
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Storage' is not registered.",
		},
		{
			name: "update a cluster from succeeded",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-removed"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								PullSecret:           `{"will":"be-kept"}`,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							IngressProfiles: []api.IngressProfile{{Name: "will-be-removed"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "will-be-removed"}},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientSecret: "will-be-kept",
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								PullSecret:           `{"will":"be-kept"}`,
								Domain:               "changed",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientSecret: "will-be-kept",
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Domain:               "changed",
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					ServicePrincipalProfile: &v20240812preview.ServicePrincipalProfile{},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "update a cluster from failed during update",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-removed"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							IngressProfiles:         []api.IngressProfile{{Name: "will-be-removed"}},
							WorkerProfiles:          []api.WorkerProfile{{Name: "will-be-removed"}},
							OperatorFlags:           api.OperatorFlags{},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateUpdating,
							LastProvisioningState:   api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain:               "changed",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Domain:               "changed",
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "update a cluster from failed during creation",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateCreating,
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name: "update a cluster from failed during deletion",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateDeleting,
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name: "patch a cluster from succeeded",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
				oc.Properties.IngressProfiles = []v20240812preview.IngressProfile{{Name: "changed"}}
				oc.Properties.WorkerProfiles = []v20240812preview.WorkerProfile{{Name: "changed"}}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain:               "changed",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							IngressProfiles: []api.IngressProfile{{Name: "changed"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "changed",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Tags:       map[string]string{"tag": "will-be-kept"},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Domain:               "changed",
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					IngressProfiles: []v20240812preview.IngressProfile{{Name: "changed"}},
					WorkerProfiles: []v20240812preview.WorkerProfile{
						{
							Name:             "changed",
							EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
						},
					},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "patch a workload identity cluster succeeded",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"disk-csi-driver":          {ResourceID: mockMiResourceId},
						"extra-new-operator":       {ResourceID: mockMiResourceId},
					},
					UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
				}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							ClusterProfile: api.ClusterProfile{
								Version: defaultVersion,
							},
							OperatorFlags: api.OperatorFlags{},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
								},
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							IngressProfiles: []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"extra-new-operator": {
										ResourceID: mockMiResourceId,
									},
								},
								UpgradeableTo: pointerutils.ToPtr(api.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
							},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Tags:       map[string]string{"tag": "will-be-kept"},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Version:              defaultVersion,
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					IngressProfiles: []v20240812preview.IngressProfile{{Name: "default"}},
					WorkerProfiles: []v20240812preview.WorkerProfile{
						{
							Name:             "default",
							EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
						},
					},

					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
					PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
							"file-csi-driver": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"cloud-controller-manager": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"ingress": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"image-registry": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"machine-api": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"cloud-network-config": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"aro-operator": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"disk-csi-driver": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"extra-new-operator": {
								ResourceID: mockMiResourceId,
							},
						},
						UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
					},
				},
			},
		},
		{
			name: "patch a workload identity cluster - can replace platform workload identities and existing clientIDs+objectIDs are removed",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"aro-operator": {ResourceID: mockMiResourceId2},
					},
				}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							ClusterProfile: api.ClusterProfile{
								Version: defaultVersion,
							},
							OperatorFlags: api.OperatorFlags{},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
								},
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							IngressProfiles: []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId2,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
								},
							},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Tags:       map[string]string{"tag": "will-be-kept"},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Version:              defaultVersion,
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					IngressProfiles: []v20240812preview.IngressProfile{{Name: "default"}},
					WorkerProfiles: []v20240812preview.WorkerProfile{
						{
							Name:             "default",
							EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
						},
					},

					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
					PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
							"file-csi-driver": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"cloud-controller-manager": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"ingress": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"image-registry": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"machine-api": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"cloud-network-config": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
							"aro-operator": {
								ResourceID: mockMiResourceId2,
							},
							"disk-csi-driver": {
								ResourceID: mockMiResourceId,
								ClientID:   mockGuid,
								ObjectID:   mockGuid,
							},
						},
					},
				},
			},
		},
		{
			name: "patch a workload identity cluster - unexpected identity provided",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"disk-csi-driver":          {ResourceID: mockMiResourceId},
						"unexpected-operator":      {ResourceID: mockMiResourceId},
					},
					UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
				}
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "default",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							ClusterProfile: api.ClusterProfile{
								Version: defaultVersion,
							},
							OperatorFlags: api.OperatorFlags{},
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
									"file-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-controller-manager": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"ingress": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"image-registry": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"machine-api": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"cloud-network-config": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"aro-operator": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
									"disk-csi-driver": {
										ResourceID: mockMiResourceId,
										ClientID:   mockGuid,
										ObjectID:   mockGuid,
									},
								},
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf(`400: PlatformWorkloadIdentityMismatch: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s or %s'. The required platform workload identities are '[aro-operator cloud-controller-manager cloud-network-config disk-csi-driver extra-new-operator file-csi-driver image-registry ingress machine-api]'`, defaultMinorVersion, getMIWIUpgradeableToVersion().MinorVersion()),
		},
		{
			name: "patch a cluster from failed during update",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							IngressProfiles:         []api.IngressProfile{{Name: "will-be-kept"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "will-be-kept",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateUpdating,
							LastProvisioningState:   api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain:               "changed",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							IngressProfiles: []api.IngressProfile{{Name: "will-be-kept"}},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:             "will-be-kept",
									EncryptionAtHost: api.EncryptionAtHostDisabled,
								},
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
								PreconfiguredNSG:       api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockGuid, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20240812preview.OpenShiftCluster{
				ID:         testdatabase.GetResourcePath(mockGuid, "resourceName"),
				Name:       "resourceName",
				Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
				SystemData: &v20240812preview.SystemData{},
				Tags:       map[string]string{"tag": "will-be-kept"},
				Properties: v20240812preview.OpenShiftClusterProperties{
					ProvisioningState: v20240812preview.ProvisioningStateUpdating,
					ClusterProfile: v20240812preview.ClusterProfile{
						Domain:               "changed",
						FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
					},
					IngressProfiles: []v20240812preview.IngressProfile{{Name: "will-be-kept"}},
					WorkerProfiles: []v20240812preview.WorkerProfile{
						{
							Name:             "will-be-kept",
							EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
						},
					},
					MasterProfile: v20240812preview.MasterProfile{
						EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					},
					NetworkProfile: v20240812preview.NetworkProfile{
						OutboundType:     v20240812preview.OutboundTypeLoadbalancer,
						PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
						LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
							ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
								Count: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "patch a cluster from failed during creation",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name: "patch a cluster from failed during deletion",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateDeleting,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name: "creating cluster failing when provided cluster resource group already contains a cluster",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile = &v20240812preview.ServicePrincipalProfile{
					ClientID: mockGuid,
				}
				oc.Properties.ClusterProfile.ResourceGroupID = fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockGuid)
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(mockGuid, "otherResourceName")),
					ClusterResourceGroupIDKey: strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockGuid)),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "otherResourceName"),
						Name: "otherResourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockGuid),
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateResourceGroup: : The provided resource group '/subscriptions/%s/resourcegroups/aro-vjb21wca' already contains a cluster.", mockGuid),
		},
		{
			name: "creating service principal cluster fails when provided client ID is not unique",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile = &v20240812preview.ServicePrincipalProfile{
					ClientID: mockGuid,
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:         strings.ToLower(testdatabase.GetResourcePath(mockGuid, "otherResourceName")),
					ClientIDKey: mockGuid,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "otherResourceName"),
						Name: "otherResourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateClientID: : The provided service principal with client ID '%s' is already in use by a cluster.", mockGuid),
		},
		{
			name: "creating workload identity cluster fails when provided cluster MSI is not unique",
			request: func(oc *v20240812preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = defaultVersion
				oc.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
						"file-csi-driver":          {ResourceID: mockMiResourceId},
						"cloud-controller-manager": {ResourceID: mockMiResourceId},
						"ingress":                  {ResourceID: mockMiResourceId},
						"image-registry":           {ResourceID: mockMiResourceId},
						"machine-api":              {ResourceID: mockMiResourceId},
						"cloud-network-config":     {ResourceID: mockMiResourceId},
						"aro-operator":             {ResourceID: mockMiResourceId},
						"disk-csi-driver":          {ResourceID: mockMiResourceId},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockGuid,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:         strings.ToLower(testdatabase.GetResourcePath(mockGuid, "otherResourceName")),
					ClientIDKey: strings.ToLower(mockMiResourceId),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockGuid, "otherResourceName"),
						Name: "otherResourceName",
						Identity: &api.ManagedServiceIdentity{
							Type: "UserAssigned",
							UserAssignedIdentities: map[string]api.UserAssignedIdentity{
								mockMiResourceId: {},
							},
						},
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version:              defaultVersion,
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
								OutboundType:           api.OutboundTypeLoadbalancer,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
							OperatorFlags: api.OperatorFlags{},
						},
					},
				})
			},
			ocpVersionsChangeFeed:  getOCPVersionsChangeFeed(),
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateClientID: : The provided user assigned identity '%s' is already in use by a cluster.", mockMiResourceId),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations().
				WithOpenShiftVersions()
			defer ti.done()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockQuotaValidator := mock_frontend.NewMockQuotaValidator(controller)
			mockQuotaValidator.EXPECT().ValidateQuota(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.quotaValidatorError).AnyTimes()

			mockSkuValidator := mock_frontend.NewMockSkuValidator(controller)

			mockSkuValidator.EXPECT().ValidateVMSku(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.skuValidatorError).AnyTimes()
			mockProvidersValidator := mock_frontend.NewMockProvidersValidator(controller)
			mockProvidersValidator.EXPECT().ValidateProviders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.providersValidatorError).AnyTimes()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, apis, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			f.quotaValidator = mockQuotaValidator
			f.skuValidator = mockSkuValidator
			f.providersValidator = mockProvidersValidator
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)
			f.ocpVersionsMu.Lock()
			f.enabledOcpVersions = tt.ocpVersionsChangeFeed
			for key, doc := range tt.ocpVersionsChangeFeed {
				if doc.Properties.Default {
					f.defaultOcpVersion = key
				}
			}
			f.ocpVersionsMu.Unlock()

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := &v20240812preview.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			requestHeaders := http.Header{
				"Content-Type": []string{"application/json"},
			}

			var internal api.OpenShiftCluster
			f.apis["2024-08-12-preview"].OpenShiftClusterConverter.ToInternal(oc, &internal)
			if internal.UsesWorkloadIdentity() {
				requestHeaders.Add(middleware.MsiIdentityURLHeader, middleware.MockIdentityURL)
				requestHeaders.Add(middleware.MsiTenantHeader, mockGuid)
			}

			resp, b, err := ti.request(method,
				"https://server"+testdatabase.GetResourcePath(mockGuid, "resourceName")+"?api-version=2024-08-12-preview",
				requestHeaders,
				oc,
			)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockGuid, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

func TestPutOrPatchOpenShiftClusterValidated(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockCurrentTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	createTime := time.Unix(199, 0)
	lastModifyTime := time.Unix(299, 0)
	newLastModifyTime := time.Unix(3000, 0)

	type test struct {
		name                   string
		request                func() *v20220401.OpenShiftCluster
		systemData             *api.SystemData
		isPatch                bool
		fixture                func(*testdatabase.Fixture)
		wantEnriched           []string
		wantSystemDataEnriched bool
		wantDocuments          func(*testdatabase.Checker)
		wantStatusCode         int
		wantResponse           *v20220401.OpenShiftCluster
		wantAsync              bool
		wantError              string
	}

	for _, tt := range []*test{
		{
			name: "PUT a cluster from succeeded does not change SystemData",
			request: func() *v20220401.OpenShiftCluster {
				return &v20220401.OpenShiftCluster{
					ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
					Name:     "resourceName",
					Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags:     map[string]string{"tag": "tag"},
					Location: "eastus",
					Properties: v20220401.OpenShiftClusterProperties{
						ClusterProfile: v20220401.ClusterProfile{
							Domain:               "example.aroapp.io",
							ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterResourceGroup", mockSubID),
							FipsValidatedModules: v20220401.FipsValidatedModulesDisabled,
						},
						MasterProfile: v20220401.MasterProfile{
							EncryptionAtHost: v20220401.EncryptionAtHostDisabled,
							VMSize:           v20220401.VMSize("Standard_D32s_v3"),
							SubnetID:         fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
						},
						ServicePrincipalProfile: &v20220401.ServicePrincipalProfile{
							ClientID:     "00000000-0000-0000-1111-000000000000",
							ClientSecret: "00000000-0000-0000-0000-000000000000",
						},
						NetworkProfile: v20220401.NetworkProfile{
							PodCIDR:     "10.0.0.0/16",
							ServiceCIDR: "10.1.0.0/16",
						},
						APIServerProfile: v20220401.APIServerProfile{
							Visibility: v20220401.VisibilityPrivate,
						},
						IngressProfiles: []v20220401.IngressProfile{
							{
								Visibility: v20220401.VisibilityPublic,
							},
						},
					},
				}
			},
			systemData: &api.SystemData{
				LastModifiedBy:     "OtherUser",
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &newLastModifyTime,
			},
			isPatch: false,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
						Location: "eastus",
						Tags:     map[string]string{"tag": "will-not-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain:               "example.aroapp.io",
								ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterResourceGroup", mockSubID),
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
								VMSize:           api.VMSize("Standard_D32s_v3"),
								SubnetID:         fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
							},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientID:     "00000000-0000-0000-1111-000000000000",
								ClientSecret: "00000000-0000-0000-0000-000000000000",
							},
							NetworkProfile: api.NetworkProfile{
								PodCIDR:     "10.0.0.0/16",
								ServiceCIDR: "10.1.0.0/16",
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPublic,
								},
							},
							OperatorFlags: api.OperatorFlags{},
						},
						SystemData: api.SystemData{
							CreatedBy:          "ExampleUser",
							CreatedByType:      api.CreatedByTypeApplication,
							CreatedAt:          &createTime,
							LastModifiedBy:     "ExampleUser",
							LastModifiedByType: api.CreatedByTypeApplication,
							LastModifiedAt:     &lastModifyTime,
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
						Location: "eastus",
						Tags:     map[string]string{"tag": "tag"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain:               "example.aroapp.io",
								ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterResourceGroup", mockSubID),
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
								VMSize:           api.VMSize("Standard_D32s_v3"),
								SubnetID:         fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
							},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientID:     "00000000-0000-0000-1111-000000000000",
								ClientSecret: "00000000-0000-0000-0000-000000000000",
							},
							NetworkProfile: api.NetworkProfile{
								PodCIDR:          "10.0.0.0/16",
								ServiceCIDR:      "10.1.0.0/16",
								OutboundType:     api.OutboundTypeLoadbalancer,
								PreconfiguredNSG: api.PreconfiguredNSGDisabled,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPublic,
								},
							},
							OperatorFlags: api.OperatorFlags{},
						},
						SystemData: api.SystemData{
							CreatedBy:          "ExampleUser",
							CreatedByType:      api.CreatedByTypeApplication,
							CreatedAt:          &createTime,
							LastModifiedBy:     "OtherUser",
							LastModifiedByType: api.CreatedByTypeApplication,
							LastModifiedAt:     &newLastModifyTime,
						},
					},
				})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &v20220401.OpenShiftCluster{
				ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
				Name:     "resourceName",
				Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
				Tags:     map[string]string{"tag": "tag"},
				Location: "eastus",
				Properties: v20220401.OpenShiftClusterProperties{
					ProvisioningState: v20220401.ProvisioningStateUpdating,
					ClusterProfile: v20220401.ClusterProfile{
						Domain:               "example.aroapp.io",
						ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterResourceGroup", mockSubID),
						FipsValidatedModules: v20220401.FipsValidatedModulesDisabled,
					},
					MasterProfile: v20220401.MasterProfile{
						EncryptionAtHost: v20220401.EncryptionAtHostDisabled,
						VMSize:           v20220401.VMSize("Standard_D32s_v3"),
						SubnetID:         fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
					},
					ServicePrincipalProfile: &v20220401.ServicePrincipalProfile{
						ClientID: "00000000-0000-0000-1111-000000000000",
					},
					NetworkProfile: v20220401.NetworkProfile{
						PodCIDR:     "10.0.0.0/16",
						ServiceCIDR: "10.1.0.0/16",
					},
					APIServerProfile: v20220401.APIServerProfile{
						Visibility: v20220401.VisibilityPrivate,
					},
					IngressProfiles: []v20220401.IngressProfile{
						{
							Visibility: v20220401.VisibilityPublic,
						},
					},
				},
				SystemData: &v20220401.SystemData{
					CreatedBy:          "ExampleUser",
					CreatedByType:      v20220401.CreatedByTypeApplication,
					CreatedAt:          &createTime,
					LastModifiedBy:     "OtherUser",
					LastModifiedByType: v20220401.CreatedByTypeApplication,
					LastModifiedAt:     &newLastModifyTime,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				enrichClusterSystemData(doc, systemData)
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)

			oc := tt.request()

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			headers := http.Header{
				"Content-Type": []string{"application/json"},
			}

			if tt.systemData != nil {
				systemData, err := json.Marshal(tt.systemData)
				if err != nil {
					t.Fatal(err)
				}
				headers["X-Ms-Arm-Resource-System-Data"] = []string{string(systemData)}
			} else {
				headers["X-Ms-Arm-Resource-System-Data"] = []string{"{}"}
			}

			resp, b, err := ti.request(method,
				"https://server"+testdatabase.GetResourcePath(mockSubID, "resourceName")+"?api-version=2022-04-01",
				headers, oc)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

func TestEnrichClusterSystemData(t *testing.T) {
	accountID1 := "00000000-0000-0000-0000-000000000001"
	accountID2 := "00000000-0000-0000-0000-000000000002"
	timestampString := "2021-01-23T12:34:54.0000000Z"
	timestamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		systemData *api.SystemData
		expected   *api.OpenShiftClusterDocument
	}{
		{
			name:       "new systemData is nil",
			systemData: nil,
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			},
		},
		{
			name: "new systemData has all fields",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID1,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID1,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "update object",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "old cluster update. Creation unknown",
			systemData: &api.SystemData{
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			}
			enrichClusterSystemData(doc, tt.systemData)

			if !reflect.DeepEqual(doc, tt.expected) {
				t.Error(cmp.Diff(doc, tt.expected))
			}
		})
	}
}

func TestValidateIdentityUrl(t *testing.T) {
	for _, tt := range []struct {
		name        string
		identityURL string
		cluster     *api.OpenShiftCluster
		expected    *api.OpenShiftCluster
		wantError   error
	}{
		{
			name:        "identity URL is empty",
			identityURL: "",
			cluster:     &api.OpenShiftCluster{},
			expected:    &api.OpenShiftCluster{},
			wantError:   errMissingIdentityParameter,
		},
		{
			name: "pass - identity URL passed",
			cluster: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{},
			},
			identityURL: "http://foo.bar",
			expected: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{
					IdentityURL: "http://foo.bar",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentityUrl(tt.cluster, tt.identityURL)
			if !errors.Is(err, tt.wantError) {
				t.Error(cmp.Diff(err, tt.wantError))
			}

			if !reflect.DeepEqual(tt.cluster, tt.expected) {
				t.Error(cmp.Diff(tt.cluster, tt.expected))
			}
		})
	}
}

func TestValidateIdentityTenantID(t *testing.T) {
	for _, tt := range []struct {
		name      string
		tenantID  string
		cluster   *api.OpenShiftCluster
		expected  *api.OpenShiftCluster
		wantError error
	}{
		{
			name:      "tenantID is empty",
			tenantID:  "",
			cluster:   &api.OpenShiftCluster{},
			expected:  &api.OpenShiftCluster{},
			wantError: errMissingIdentityParameter,
		},
		{
			name: "pass - tenantID passed",
			cluster: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{},
			},
			tenantID: "bogus",
			expected: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{
					TenantID: "bogus",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentityTenantID(tt.cluster, tt.tenantID)
			if !errors.Is(err, tt.wantError) {
				t.Error(cmp.Diff(err, tt.wantError))
			}

			if !reflect.DeepEqual(tt.cluster, tt.expected) {
				t.Error(cmp.Diff(tt.cluster, tt.expected))
			}
		})
	}
}

func getMIWIUpgradeableToVersion() *version.Version {
	ver := version.DefaultInstallStream.Version.V
	return version.NewVersion(ver[0], ver[1]+1, ver[2])
}

func getOCPVersionsChangeFeed() map[string]*api.OpenShiftVersion {
	return map[string]*api.OpenShiftVersion{
		defaultVersion: {
			Properties: api.OpenShiftVersionProperties{
				Version: defaultVersion,
				Enabled: true,
				Default: true,
			},
		},
		getMIWIUpgradeableToVersion().String(): {
			Properties: api.OpenShiftVersionProperties{
				Version: getMIWIUpgradeableToVersion().String(),
				Enabled: true,
				Default: false,
			},
		},
	}
}

func getPlatformWorkloadIdentityRolesChangeFeed() map[string]*api.PlatformWorkloadIdentityRoleSet {
	return map[string]*api.PlatformWorkloadIdentityRoleSet{
		defaultMinorVersion: {
			Properties: api.PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: defaultMinorVersion,
				PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
					{OperatorName: "cloud-controller-manager"},
					{OperatorName: "ingress"},
					{OperatorName: "machine-api"},
					{OperatorName: "disk-csi-driver"},
					{OperatorName: "cloud-network-config"},
					{OperatorName: "image-registry"},
					{OperatorName: "file-csi-driver"},
					{OperatorName: "aro-operator"},
				},
			},
		},
		getMIWIUpgradeableToVersion().MinorVersion(): {
			Properties: api.PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: getMIWIUpgradeableToVersion().MinorVersion(),
				PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
					{OperatorName: "cloud-controller-manager"},
					{OperatorName: "ingress"},
					{OperatorName: "machine-api"},
					{OperatorName: "disk-csi-driver"},
					{OperatorName: "cloud-network-config"},
					{OperatorName: "image-registry"},
					{OperatorName: "file-csi-driver"},
					{OperatorName: "aro-operator"},
					{OperatorName: "extra-new-operator"},
				},
			},
		},
	}
}

// TestConversion_PreconfiguredNSG checks that converting from the internal
// api.OpenShiftCluster to the external admin.OpenShiftCluster preserves or
// defaults the PreconfiguredNSG field correctly.
func TestConversion_PreconfiguredNSG(t *testing.T) {
	// Grab the existing converter from the "admin" API version.
	// This is what actually does the internal->external conversion.
	converter := api.APIs["admin"].OpenShiftClusterConverter

	tests := []struct {
		name     string
		input    *api.OpenShiftCluster
		expected admin.PreconfiguredNSG
	}{
		{
			name: "preconfiguredNSG is correctly preserved as Enabled",
			input: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						PreconfiguredNSG: api.PreconfiguredNSGEnabled,
					},
				},
			},
			expected: admin.PreconfiguredNSGEnabled,
		},
		{
			name: "preconfiguredNSG defaults to Disabled when missing",
			input: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			expected: admin.PreconfiguredNSGDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalObj := converter.ToExternal(tt.input)
			oc, ok := externalObj.(*admin.OpenShiftCluster)
			if !ok {
				t.Fatalf("expected *admin.OpenShiftCluster, got %T", externalObj)
			}
			if oc.Properties.NetworkProfile.PreconfiguredNSG != tt.expected {
				t.Errorf("expected %v, got %v",
					tt.expected, oc.Properties.NetworkProfile.PreconfiguredNSG)
			}
		})
	}
}

// TestRegression_PreconfiguredNSGField ensures we don't regress on preserving
// the PreconfiguredNSG field in future changes to conversion logic.
func TestRegression_PreconfiguredNSGField(t *testing.T) {
	converter := api.APIs["admin"].OpenShiftClusterConverter

	input := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			NetworkProfile: api.NetworkProfile{
				PreconfiguredNSG: api.PreconfiguredNSGEnabled,
			},
		},
	}

	externalObj := converter.ToExternal(input)
	oc, ok := externalObj.(*admin.OpenShiftCluster)
	if !ok {
		t.Fatalf("expected *admin.OpenShiftCluster, got %T", externalObj)
	}

	if oc.Properties.NetworkProfile.PreconfiguredNSG != admin.PreconfiguredNSGEnabled {
		t.Errorf("expected PreconfiguredNSG=Enabled, got %v",
			oc.Properties.NetworkProfile.PreconfiguredNSG)
	}
}
