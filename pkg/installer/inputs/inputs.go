package inputs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/yaml"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

// InstallerInputs contains all the inputs needed to run the OpenShift installer
// in any execution backend (Hive, AKSJob, Podman)
type InstallerInputs struct {
	// Cluster and subscription documents
	ClusterJSON      []byte
	SubscriptionJSON []byte

	// Installer images
	InstallerPullspec string
	OpenShiftPullspec string
	PullSecretJSON    []byte

	// Credentials for target cluster
	// Only one of ServicePrincipalJSON or BoundSASigningKey should be set
	ServicePrincipalJSON []byte // For service principal clusters
	BoundSASigningKey    []byte // For managed identity clusters

	// Custom manifests
	CustomManifests map[string][]byte

	// Proxy certificates (development only)
	ProxyCert       []byte
	ProxyClientCert []byte
	ProxyClientKey  []byte

	// Environment variables
	EnvironmentVariables map[string]string

	// Metadata
	ClusterUUID               string
	ResourceID                string
	SubscriptionID            string
	CorrelationData           *api.CorrelationData
	Location                  string
	Domain                    string
	Namespace                 string
	ExecutionID               string
	JobName                   string
	Timeout                   time.Duration
	IsDevelopmentMode         bool
	InstallerIdentityClientID string
}

// Builder constructs InstallerInputs from cluster and subscription documents
type Builder struct {
	env env.Interface
}

// NewBuilder creates a new installer inputs builder
func NewBuilder(env env.Interface) *Builder {
	return &Builder{
		env: env,
	}
}

// Build creates InstallerInputs from the provided cluster, subscription, and version
func (b *Builder) Build(
	doc *api.OpenShiftClusterDocument,
	sub *api.SubscriptionDocument,
	version *api.OpenShiftVersion,
	customManifests map[string]kruntime.Object,
) (*InstallerInputs, error) {
	inputs := &InstallerInputs{
		ClusterUUID:          doc.ID,
		ResourceID:           doc.OpenShiftCluster.ID,
		SubscriptionID:       sub.ID,
		CorrelationData:      doc.CorrelationData,
		Location:             doc.OpenShiftCluster.Location,
		InstallerPullspec:    version.Properties.InstallerPullspec,
		OpenShiftPullspec:    version.Properties.OpenShiftPullspec,
		CustomManifests:      make(map[string][]byte),
		EnvironmentVariables: make(map[string]string),
		Timeout:              60 * time.Minute,
		IsDevelopmentMode:    b.env.IsLocalDevelopmentMode(),
	}

	// Serialize cluster document
	clusterJSON, err := json.Marshal(doc.OpenShiftCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster document: %w", err)
	}
	inputs.ClusterJSON = clusterJSON

	// Serialize subscription document
	subJSON, err := json.Marshal(sub.Subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription document: %w", err)
	}
	inputs.SubscriptionJSON = subJSON

	pullSecretJSON, err := pullsecret.Build(doc.OpenShiftCluster, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build installer pull secret: %w", err)
	}
	inputs.PullSecretJSON = []byte(pullSecretJSON)

	// Set domain
	inputs.Domain = doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(inputs.Domain, '.') {
		domainSuffix := os.Getenv("DOMAIN_NAME")
		if domainSuffix != "" {
			inputs.Domain += "." + domainSuffix
		}
	}

	// Handle credentials based on cluster type
	if doc.OpenShiftCluster.UsesWorkloadIdentity() {
		boundKey, err := b.getBoundSASigningKey(doc.OpenShiftCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to get bound SA signing key: %w", err)
		}
		inputs.BoundSASigningKey = boundKey
	} else {
		// Service principal cluster - add SP credentials
		spJSON, err := b.getServicePrincipalJSON(doc.OpenShiftCluster, sub)
		if err != nil {
			return nil, fmt.Errorf("failed to get service principal credentials: %w", err)
		}
		inputs.ServicePrincipalJSON = spJSON
	}

	// Convert custom manifests to bytes
	for key, obj := range customManifests {
		manifestBytes, err := yaml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal manifest %s: %w", key, err)
		}
		inputs.CustomManifests[key] = manifestBytes
	}

	// Set environment variables
	inputs.EnvironmentVariables["ARO_UUID"] = doc.ID
	inputs.EnvironmentVariables["OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE"] = version.Properties.OpenShiftPullspec
	inputs.EnvironmentVariables["OPENSHIFT_INSTALL_INVOKER"] = "hive" // Keep for compatibility

	if inputs.IsDevelopmentMode {
		inputs.EnvironmentVariables["ARO_RP_MODE"] = "development"
		err := b.addDevelopmentEnvironment(inputs)
		if err != nil {
			return nil, err
		}
	} else {
		err := b.addProductionEnvironment(inputs)
		if err != nil {
			return nil, err
		}
	}

	return inputs, nil
}

// getServicePrincipalJSON creates the osServicePrincipal.json for service principal clusters
func (b *Builder) getServicePrincipalJSON(oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) ([]byte, error) {
	if oc.Properties.ServicePrincipalProfile == nil {
		return nil, fmt.Errorf("service principal profile is nil")
	}

	spData := map[string]string{
		"subscriptionId": sub.ID,
		"tenantId":       sub.Subscription.Properties.TenantID,
		"clientId":       oc.Properties.ServicePrincipalProfile.ClientID,
		"clientSecret":   string(oc.Properties.ServicePrincipalProfile.ClientSecret),
	}

	return json.Marshal(spData)
}

// getBoundSASigningKey retrieves the bound service account signing key for workload identity clusters
func (b *Builder) getBoundSASigningKey(oc *api.OpenShiftCluster) ([]byte, error) {
	if oc.Properties.ClusterProfile.BoundServiceAccountSigningKey == nil {
		return nil, fmt.Errorf("properties.clusterProfile.boundServiceAccountSigningKey not set")
	}

	return []byte(*oc.Properties.ClusterProfile.BoundServiceAccountSigningKey), nil
}

// addDevelopmentEnvironment adds development-specific environment variables
func (b *Builder) addDevelopmentEnvironment(inputs *InstallerInputs) error {
	devEnvVars := []string{
		"AZURE_FP_CLIENT_ID",
		"AZURE_RP_CLIENT_ID",
		"AZURE_RP_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"DOMAIN_NAME",
		"KEYVAULT_PREFIX",
		"LOCATION",
		"PROXY_HOSTNAME",
		"PULL_SECRET",
		"RESOURCEGROUP",
	}

	for _, envvar := range devEnvVars {
		val := os.Getenv(envvar)
		if val != "" {
			inputs.EnvironmentVariables["ARO_"+envvar] = val
		}
	}

	// Load proxy certificates for development
	err := b.loadProxyCertificates(inputs)
	if err != nil {
		return fmt.Errorf("failed to load proxy certificates: %w", err)
	}

	return nil
}

// addProductionEnvironment adds production-specific environment variables
func (b *Builder) addProductionEnvironment(inputs *InstallerInputs) error {
	prodEnvVars := []string{
		"AZURE_FP_CLIENT_ID",
		"CLUSTER_MDSD_ACCOUNT",
		"CLUSTER_MDSD_CONFIG_VERSION",
		"CLUSTER_MDSD_NAMESPACE",
		"DOMAIN_NAME",
		"GATEWAY_DOMAINS",
		"GATEWAY_RESOURCEGROUP",
		"KEYVAULT_PREFIX",
		"MDSD_ENVIRONMENT",
		"ACR_RESOURCE_ID",
	}

	for _, envvar := range prodEnvVars {
		inputs.EnvironmentVariables["ARO_"+envvar] = os.Getenv(envvar)
	}

	for _, envvar := range []string{
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"LOCATION",
		"RESOURCEGROUP",
	} {
		if value := os.Getenv(envvar); value != "" {
			inputs.EnvironmentVariables["ARO_"+envvar] = value
		}
	}

	inputs.InstallerIdentityClientID = os.Getenv("ARO_INSTALLER_IDENTITY_CLIENT_ID")
	if inputs.InstallerIdentityClientID == "" {
		return fmt.Errorf("ARO_INSTALLER_IDENTITY_CLIENT_ID must be set for AKS Job installation in production")
	}

	return nil
}

// loadProxyCertificates loads proxy certificates for development mode
func (b *Builder) loadProxyCertificates(inputs *InstallerInputs) error {
	basepath := os.Getenv("ARO_CHECKOUT_PATH")
	if basepath == "" {
		// Assume we are running from an ARO-RP checkout
		_, curmod, _, _ := runtime.Caller(0)
		var err error
		basepath, err = filepath.Abs(filepath.Join(filepath.Dir(curmod), "../../.."))
		if err != nil {
			return err
		}
	}

	proxyCert, err := os.ReadFile(path.Join(basepath, "secrets/proxy.crt"))
	if err != nil {
		return err
	}
	inputs.ProxyCert = proxyCert

	proxyClientCert, err := os.ReadFile(path.Join(basepath, "secrets/proxy-client.crt"))
	if err != nil {
		return err
	}
	inputs.ProxyClientCert = proxyClientCert

	proxyClientKey, err := os.ReadFile(path.Join(basepath, "secrets/proxy-client.key"))
	if err != nil {
		return err
	}
	inputs.ProxyClientKey = proxyClientKey

	return nil
}
