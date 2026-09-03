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

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

// InstallerInputs contains all the inputs needed to run the OpenShift installer
// in any execution backend (Hive, AKSJob, Podman)
type InstallerInputs struct {
	// Cluster and subscription documents
	ClusterJSON      []byte
	SubscriptionJSON []byte

	// Installer images
	InstallerPullspec   string
	OpenShiftPullspec   string
	InstallerPullSecret *PullSecret

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
	ClusterUUID       string
	ResourceID        string
	SubscriptionID    string
	CorrelationData   *api.CorrelationData
	Location          string
	Domain            string
	Namespace         string
	ExecutionID       string
	Timeout           time.Duration
	IsDevelopmentMode bool
}

// PullSecret represents container registry credentials
type PullSecret struct {
	Username string
	Password string
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
		// Managed identity cluster - add bound SA signing key
		if doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile != nil {
			boundKey, err := b.getBoundSASigningKey(doc.OpenShiftCluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get bound SA signing key: %w", err)
			}
			inputs.BoundSASigningKey = boundKey
		}
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
		manifestBytes, err := json.Marshal(obj)
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

	spData := map[string]interface{}{
		"subscriptionId":  sub.ID,
		"tenantId":        sub.Subscription.Properties.TenantID,
		"aadClientId":     oc.Properties.ServicePrincipalProfile.ClientID,
		"aadClientSecret": string(oc.Properties.ServicePrincipalProfile.ClientSecret),
	}

	return json.Marshal(spData)
}

// getBoundSASigningKey retrieves the bound service account signing key for workload identity clusters
func (b *Builder) getBoundSASigningKey(oc *api.OpenShiftCluster) ([]byte, error) {
	// This would need to be implemented based on where the signing key is stored
	// For now, return a placeholder
	// TODO: Implement actual key retrieval from KeyVault or cluster MSI store
	return nil, fmt.Errorf("bound SA signing key retrieval not yet implemented")
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
	// Production environment variables would be injected via KeyVault or similar
	// For now, this is a placeholder
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
