package cloudproviderconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

const (
	ControllerName = "CloudProviderConfig"
)

var cloudProviderConfigName = types.NamespacedName{Name: "cloud-provider-config", Namespace: "openshift-config"}

// / This is a very old version of the CloudProviderConfig struct from k8s upstream
// / OpenShift diverged from the upstream cluster API and we have a depreciated version of this struct
// / I was unable to use the recent version of this struct in the upstream k8s Azure Cloud Provider
// / This version of the struct can be found here:
// / https://github.com/kubernetes/kubernetes/blob/v1.13.5/pkg/cloudprovider/providers/azure/azure.go#L81
type azCloudProviderConfig struct {
	// The cloud environment identifier. Takes values from https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	Cloud string `json:"cloud" yaml:"cloud"`
	// The AAD Tenant ID for the Subscription that the cluster is deployed in
	TenantID string `json:"tenantId" yaml:"tenantId"`
	// The ClientID for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientID string `json:"aadClientId" yaml:"aadClientId"`
	// The ClientSecret for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	// The path of a client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPath string `json:"aadClientCertPath" yaml:"aadClientCertPath"`
	// The password of the client certificate for an AAD application with RBAC access to talk to Azure RM APIs
	AADClientCertPassword string `json:"aadClientCertPassword" yaml:"aadClientCertPassword"`
	// Use managed service identity for the virtual machine to access Azure ARM APIs
	UseManagedIdentityExtension bool `json:"useManagedIdentityExtension" yaml:"useManagedIdentityExtension"`
	// UserAssignedIdentityID contains the Client ID of the user assigned MSI which is assigned to the underlying VMs. If empty the user assigned identity is not used.
	// More details of the user assigned identity can be found at: https://docs.microsoft.com/en-us/azure/active-directory/managed-service-identity/overview
	// For the user assigned identity specified here to be used, the UseManagedIdentityExtension has to be set to true.
	UserAssignedIdentityID string `json:"userAssignedIdentityID" yaml:"userAssignedIdentityID"`
	// The ID of the Azure Subscription that the cluster is deployed in
	SubscriptionID string `json:"subscriptionId" yaml:"subscriptionId"`
	// ResourceManagerEndpoint is the cloud's resource manager endpoint. If set, cloud provider queries this endpoint
	// in order to generate an autorest.Environment instance instead of using one of the pre-defined Environments.
	ResourceManagerEndpoint string `json:"resourceManagerEndpoint,omitempty" yaml:"resourceManagerEndpoint,omitempty"`
	// The name of the resource group that the cluster is deployed in
	ResourceGroup string `json:"resourceGroup" yaml:"resourceGroup"`
	// The location of the resource group that the cluster is deployed in
	Location string `json:"location" yaml:"location"`
	// The name of the VNet that the cluster is deployed in
	VnetName string `json:"vnetName" yaml:"vnetName"`
	// The name of the resource group that the Vnet is deployed in
	VnetResourceGroup string `json:"vnetResourceGroup" yaml:"vnetResourceGroup"`
	// The name of the subnet that the cluster is deployed in
	SubnetName string `json:"subnetName" yaml:"subnetName"`
	// The name of the security group attached to the cluster's subnet
	SecurityGroupName string `json:"securityGroupName" yaml:"securityGroupName"`
	// (Optional in 1.6) The name of the route table attached to the subnet that the cluster is deployed in
	RouteTableName string `json:"routeTableName" yaml:"routeTableName"`
	// (Optional) The name of the availability set that should be used as the load balancer backend
	// If this is set, the Azure cloudprovider will only add nodes from that availability set to the load
	// balancer backend pool. If this is not set, and multiple agent pools (availability sets) are used, then
	// the cloudprovider will try to add all nodes to a single backend pool which is forbidden.
	// In other words, if you use multiple agent pools (availability sets), you MUST set this field.
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName" yaml:"primaryAvailabilitySetName"`
	// The type of azure nodes. Candidate values are: vmss and standard.
	// If not set, it will be default to standard.
	VMType string `json:"vmType" yaml:"vmType"`
	// The name of the scale set that should be used as the load balancer backend.
	// If this is set, the Azure cloudprovider will only add nodes from that scale set to the load
	// balancer backend pool. If this is not set, and multiple agent pools (scale sets) are used, then
	// the cloudprovider will try to add all nodes to a single backend pool which is forbidden.
	// In other words, if you use multiple agent pools (scale sets), you MUST set this field.
	PrimaryScaleSetName string `json:"primaryScaleSetName" yaml:"primaryScaleSetName"`
	// Enable exponential backoff to manage resource request retries
	CloudProviderBackoff bool `json:"cloudProviderBackoff" yaml:"cloudProviderBackoff"`
	// Backoff retry limit
	CloudProviderBackoffRetries int `json:"cloudProviderBackoffRetries" yaml:"cloudProviderBackoffRetries"`
	// Backoff exponent
	CloudProviderBackoffExponent float64 `json:"cloudProviderBackoffExponent" yaml:"cloudProviderBackoffExponent"`
	// Backoff duration
	CloudProviderBackoffDuration int `json:"cloudProviderBackoffDuration" yaml:"cloudProviderBackoffDuration"`
	// Backoff jitter
	CloudProviderBackoffJitter float64 `json:"cloudProviderBackoffJitter" yaml:"cloudProviderBackoffJitter"`
	// Enable rate limiting
	CloudProviderRateLimit bool `json:"cloudProviderRateLimit" yaml:"cloudProviderRateLimit"`
	// Rate limit QPS (Read)
	CloudProviderRateLimitQPS float32 `json:"cloudProviderRateLimitQPS" yaml:"cloudProviderRateLimitQPS"`
	// Rate limit Bucket Size
	CloudProviderRateLimitBucket int `json:"cloudProviderRateLimitBucket" yaml:"cloudProviderRateLimitBucket"`
	// Rate limit QPS (Write)
	CloudProviderRateLimitQPSWrite float32 `json:"cloudProviderRateLimitQPSWrite" yaml:"cloudProviderRateLimitQPSWrite"`
	// Rate limit Bucket Size
	CloudProviderRateLimitBucketWrite int `json:"cloudProviderRateLimitBucketWrite" yaml:"cloudProviderRateLimitBucketWrite"`

	// Use instance metadata service where possible
	UseInstanceMetadata bool `json:"useInstanceMetadata" yaml:"useInstanceMetadata"`

	// Sku of Load Balancer and Public IP. Candidate values are: basic and standard.
	// If not set, it will be default to basic.
	LoadBalancerSku string `json:"loadBalancerSku" yaml:"loadBalancerSku"`
	// ExcludeMasterFromStandardLB excludes master nodes from standard load balancer.
	// If not set, it will be default to true.
	ExcludeMasterFromStandardLB *bool `json:"excludeMasterFromStandardLB" yaml:"excludeMasterFromStandardLB"`
	// DisableOutboundSNAT disables the outbound SNAT for public load balancer rules.
	// It should only be set when loadBalancerSku is standard. If not set, it will be default to false.
	DisableOutboundSNAT *bool `json:"disableOutboundSNAT" yaml:"disableOutboundSNAT"`

	// Maximum allowed LoadBalancer Rule Count is the limit enforced by Azure Load balancer
	MaximumLoadBalancerRuleCount int `json:"maximumLoadBalancerRuleCount" yaml:"maximumLoadBalancerRuleCount"`
}

// CloudProviderConfigReconciler reconciles the openshift-config/cloud-provider-config ConfigMap
type CloudProviderConfigReconciler struct {
	base.AROController
}

func NewReconciler(Log *logrus.Entry, client client.Client) *CloudProviderConfigReconciler {
	return &CloudProviderConfigReconciler{
		AROController: base.AROController{
			Log:    Log,
			Client: client,
			Name:   ControllerName,
		},
	}
}

// Reconcile makes sure that the cloud-provider-config is healthy
func (r *CloudProviderConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.Debug("reconcile ConfigMap openshift-config/cloud-provider-config")

	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.CloudProviderConfigEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	return reconcile.Result{}, r.updateCloudProviderConfig(ctx)
}

// SetupWithManager setup our manager
func (r *CloudProviderConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("starting cloud-provider-config controller")

	cloudProviderConfigPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == cloudProviderConfigName.Name && o.GetNamespace() == cloudProviderConfigName.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(
			&corev1.ConfigMap{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(cloudProviderConfigPredicate),
		).
		Named(ControllerName).
		Complete(r)
}

// GetDisableOutboundSNAT Returns the value of disableOutboundSNAT from the Config
func GetDisableOutboundSNAT(jsonConfig string) (bool, error) {
	cpc, err := unmarshalCloudProviderConfigData(jsonConfig)
	if err != nil {
		return false, err
	}
	return *cpc.DisableOutboundSNAT, nil
}

func (r *CloudProviderConfigReconciler) getCloudProviderConfigFromCluster(ctx context.Context) (*corev1.ConfigMap, string, error) {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, cloudProviderConfigName, cm)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Debug("the ConfigMap cloud-provider-config was not found in the openshift-config namespace")
		}
		return nil, "", err
	}
	jsonConfig, ok := cm.Data["config"]
	if !ok {
		return nil, "", fmt.Errorf("field config in ConfigMap openshift-config/cloud-provider-config is missing")
	}
	return cm, jsonConfig, nil
}

func unmarshalCloudProviderConfigData(jsonConfig string) (*azCloudProviderConfig, error) {
	var cpc azCloudProviderConfig
	err := json.Unmarshal([]byte(jsonConfig), &cpc)
	if err != nil {
		return nil, err
	}
	return &cpc, nil
}

func marshalCloudProvderConfigData(cpc *azCloudProviderConfig) (string, error) {
	jsonStringByte, err := json.Marshal(cpc)
	if err != nil {
		return "", err
	}
	return string(jsonStringByte), err
}

func (r *CloudProviderConfigReconciler) updateCloudProviderConfig(ctx context.Context) error {
	r.Log.Debug("checking openshift-config/cloud-provider-config")

	cm, jsonConfig, err := r.getCloudProviderConfigFromCluster(ctx)
	if err != nil {
		return err
	}

	cpc, err := unmarshalCloudProviderConfigData(jsonConfig)
	if err != nil {
		return err
	}

	if cpc.DisableOutboundSNAT != nil && !*cpc.DisableOutboundSNAT {
		r.Log.Info("updating openshift-config/cloud-provider-config disableOutboundSNAT from false to true")
		*cpc.DisableOutboundSNAT = true
	} else if cpc.DisableOutboundSNAT == nil {
		r.Log.Info("updating openshift-config/cloud-provider-config disableOutboundSNAT from nil to true")
		truePointer := true
		cpc.DisableOutboundSNAT = &truePointer
	} else {
		r.Log.Debug("openshift-config/cloud-provider-config disableOutboundSNAT is set to true no changes needed")
		return nil
	}

	cm.Data["config"], err = marshalCloudProvderConfigData(cpc)
	if err != nil {
		return err
	}

	return r.Client.Update(ctx, cm)
}
