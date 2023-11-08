package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/google/go-cmp/cmp/cmpopts"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	location                 = "eastus"
	subscriptionId           = "0000000-0000-0000-0000-000000000000"
	rpSubscriptionId         = "9999999-9999-9999-9999-999999999999"
	rpResourceGroup          = "aro-" + location
	gwyResourceGroup         = "aro-gwy-" + location
	managedResourceGroupName = "aro-iljrzb5a"
	managedResourceGroupId   = "/subscriptions/" + subscriptionId + "/resourceGroups/" + managedResourceGroupName
	clusterName              = "test-cluster"
	clusterResourceId        = "/subscriptions/" + subscriptionId + "/resourceGroups/test-group/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + clusterName
	infraId                  = "abcde"
	vnetResourceGroup        = "vnet-rg"
	vnetName                 = "vnet"
	subnetNameWorker         = "worker"
	subnetNameMaster         = "master"

	storageSuffix              = "random-suffix"
	clusterStorageAccountName  = "cluster" + storageSuffix
	registryStorageAccountName = "image-registry-account"

	masterSubnetId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
	workerSubnetId = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
	rpPeSubnetId   = "/subscriptions/" + rpSubscriptionId + "/resourceGroups/" + rpResourceGroup + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"
	rpSubnetId     = "/subscriptions/" + rpSubscriptionId + "/resourceGroups/" + rpResourceGroup + "/providers/Microsoft.Network/virtualNetworks/rp-vnet/subnets/rp-subnet"
	gwySubnetId    = "/subscriptions/" + rpSubscriptionId + "/resourceGroups/" + gwyResourceGroup + "/providers/Microsoft.Network/virtualNetworks/gateway-vnet/subnets/gateway-subnet"

	serviceSubnets = []string{rpPeSubnetId, rpSubnetId, gwySubnetId}

	cmpoptsSortStringSlices = cmpopts.SortSlices(func(a, b string) bool { return a < b })
)

func TestReconcile(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	errUnknown := fmt.Errorf("unknown err")

	for _, tt := range []struct {
		name                               string
		instance                           func(*arov1alpha1.Cluster)
		operatorFlag                       bool
		fakeCheckClusterSubnetsToReconcile func(ctx context.Context, clusterSubnets []string) ([]string, error)
		fakeReconcileAccounts              func(ctx context.Context, subnets, storageAccounts []string) error
		wantRequeueAfter                   time.Duration
		wantErr                            string
	}{
		{
			name: "no cluster object - returns error",
			instance: func(c *arov1alpha1.Cluster) {
				c.ObjectMeta = metav1.ObjectMeta{}
			},
			wantErr: `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "controller disabled - does nothing",
		},
		{
			name:         "invalid resource id - returns error",
			operatorFlag: true,
			instance: func(c *arov1alpha1.Cluster) {
				c.Spec.ResourceID = "invalid resource id"
			},
			wantErr: "parsing failed for invalid resource id. Invalid resource Id format",
		},
		{
			name:         "correct prerequisites - works as expected",
			operatorFlag: true,
		},
		{
			name:         "error during subnet checks - returns direct error",
			operatorFlag: true,
			fakeCheckClusterSubnetsToReconcile: func(ctx context.Context, subnets []string) ([]string, error) {
				return nil, errUnknown
			},
			wantErr: errUnknown.Error(),
		},
		{
			name:         "error during account reconciliation - returns direct error",
			operatorFlag: true,
			fakeReconcileAccounts: func(ctx context.Context, subnets []string, storageAccounts []string) error {
				return errUnknown
			},
			wantErr: errUnknown.Error(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			instance := getValidClusterInstance(tt.operatorFlag)
			if tt.instance != nil {
				tt.instance(instance)
			}

			rc := &imageregistryv1.Config{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: imageregistryv1.ImageRegistrySpec{
					Storage: imageregistryv1.ImageRegistryConfigStorage{
						Azure: &imageregistryv1.ImageRegistryConfigStorageAzure{
							AccountName: registryStorageAccountName,
						},
					},
				},
			}

			azCredSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterauthorizer.AzureCredentialSecretName,
					Namespace: clusterauthorizer.AzureCredentialSecretNameSpace,
				},
				Data: map[string][]byte{
					"azure_client_id":     []byte("fake_client_id"),
					"azure_client_secret": []byte("fake_client_secret"),
					"azure_tenant_id":     []byte("fake-tenant-id.example.com"),
				},
			}

			clientFake := fake.NewClientBuilder().
				WithObjects(instance).
				WithObjects(rc).
				WithObjects(azCredSecret).
				WithLists(getValidMachines()).
				Build()

			fakeNewManager := func(
				log *logrus.Entry,
				gotLocation, gotSubscriptionID, gotResourceGroup string,
				azenv azureclient.AROEnvironment, authorizer autorest.Authorizer,
			) manager {
				t.Helper()

				if gotLocation != location {
					t.Errorf("wanted location %s but got %s", location, gotLocation)
				}
				if gotSubscriptionID != subscriptionId {
					t.Errorf("wanted subscriptionId %s but got %s", subscriptionId, gotSubscriptionID)
				}
				if gotResourceGroup != managedResourceGroupName {
					t.Errorf("wanted resource group %s but got %s", managedResourceGroupName, gotResourceGroup)
				}

				return &fakeManager{
					fakeCheckClusterSubnetsToReconcile: tt.fakeCheckClusterSubnetsToReconcile,
					fakeReconcileAccounts:              tt.fakeReconcileAccounts,
				}
			}

			r := Reconciler{
				log:        log,
				client:     clientFake,
				newManager: fakeNewManager,
			}

			result, err := r.Reconcile(context.Background(), reconcile.Request{})

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if result.RequeueAfter != tt.wantRequeueAfter {
				t.Errorf("wanted requeue after %v but got %v", tt.wantRequeueAfter, result.RequeueAfter)
			}
		})
	}
}

func getValidClusterInstance(operatorFlag bool) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:             clusterResourceId,
			ClusterResourceGroupID: managedResourceGroupId,
			AZEnvironment:          "AzurePublicCloud",
			Location:               location,
			StorageSuffix:          storageSuffix,
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: strconv.FormatBool(operatorFlag),
			},
			ServiceSubnets: serviceSubnets,
		},
	}
}

func getValidMachines() *machinev1beta1.MachineList {
	list := &machinev1beta1.MachineList{
		Items: []machinev1beta1.Machine{},
	}

	for i := 0; i < 3; i++ {
		list.Items = append(list.Items,
			machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-master-%d", clusterName, infraId, i),
					Namespace: "openshift-machine-api",
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-machine-role": "master",
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(fmt.Sprintf(
								`{"resourceGroup":"%s","networkResourceGroup":"%s","vnet":"%s","subnet":"%s"}`,
								managedResourceGroupName, vnetResourceGroup, vnetName, subnetNameMaster,
							)),
						},
					},
				},
			},
			machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-worker-%s%d-asdfg", clusterName, infraId, location, i),
					Namespace: "openshift-machine-api",
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-machine-role": "worker",
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &kruntime.RawExtension{
							Raw: []byte(fmt.Sprintf(
								`{"resourceGroup":"%s","networkResourceGroup":"%s","vnet":"%s","subnet":"%s"}`,
								managedResourceGroupName, vnetResourceGroup, vnetName, subnetNameWorker,
							)),
						},
					},
				},
			},
		)
	}

	return list
}

type fakeManager struct {
	fakeCheckClusterSubnetsToReconcile func(ctx context.Context, clusterSubnets []string) ([]string, error)
	fakeReconcileAccounts              func(ctx context.Context, subnets, storageAccounts []string) error
}

func (f *fakeManager) checkClusterSubnetsToReconcile(ctx context.Context, clusterSubnets []string) ([]string, error) {
	if f.fakeCheckClusterSubnetsToReconcile == nil {
		return clusterSubnets, nil
	}
	return f.fakeCheckClusterSubnetsToReconcile(ctx, clusterSubnets)
}

func (f *fakeManager) reconcileAccounts(ctx context.Context, subnets, storageAccounts []string) error {
	if f.fakeReconcileAccounts == nil {
		return nil
	}
	return f.fakeReconcileAccounts(ctx, subnets, storageAccounts)
}
