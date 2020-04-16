package testconfig

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/zhuoli/ARO-RP/test/e2e/helpers/testrandom"
)

var instance *Config
var instanceMutex sync.Mutex

// Config is a struct which holds test execution configuration. The ruling instance is a private
// singleton of the testconfig package, however the struct type is exported to allow for specifying
// test-specific overrides in Load
type Config struct {
	Cloud      string // the Cloud to target: 'public', 'usgovernment', 'china' or 'german'
	RpEndpoint string // used when private rp is running either locally or on a specific IP (OSA)

	Regions []string // a list of regions to choose from for cluster deployment location

	DeletionDueTime    string        // deletion due time of the resources created in this test run.
	EventuallyTimeout  time.Duration // duration after which to timeout eventually block
	EventuallyInterval time.Duration // duration after which to poll in eventually block

	ClusterType string // Either OSA or AKS

	OSAConfig OSAConfig // OSA Specific configuration

	CleanUpCondition CleanUpConditionEnum // values to choose from: never, failure, success, always
	ClusterVMSize    string               // the vm size use to create the cluster
	OsDiskSizeGB     int32

	ClusterNodeCount int // number of nodes in the cluster created by the tests
	ScaleMinCount    int // number of nodes to scale down to
	ScaleMaxCount    int // number of nodes to scale up to

	AdminEndpointFormat string //format that accepts one parameter of region to form the endpoint to call for admin api calls

	SuiteName string // Save Suite Name config for future validation
}

// OSAConfig is a struct containing specific OSA configuration
type OSAConfig struct {
	OSAKV                             string // OSA regional KeyVault used to generated TLS CA Signed certificates
	OSAAdminKV                        string // OSA Admin KV used to access certificates needed for admin update operations
	OSAAdminEndpoint                  string // OSA Admin Endpoint
	ValidateListByRegionStrategy      string // specify the strategy used to run ListClustersByRegion, valid inputs are: none, partial, full
	ValidatorOSATLS                   bool   // specify yes or not if the TLS Validator has to be run
	ValidatorOSAPluginTLS             bool   // specify yes or not if the TLS Plugin Validator has to be run. We are supposed to run it only in INT and Canary
	RPCredsAppID                      string // OSA RP Creds App ID used to read cluster KV
	RPCredsSecretPath                 string // OSA RP Creds Secret Path used to read cluster KV
	OSAPublisherTenantID              string // OSA Publisher Tenant ID
	StorageAccountNameClusterConfig   string // storage account name for test storage for e2e tests
	StorageAccountSecretClusterConfig string // storage account secret kv uri for e2e tests
	EnableDeepTests                   bool   // set to true to enable deep cluster tests using os cluster APIs
	EnableAdminAsyncGenevaActionTest  bool   // set to true to enable admin async geneva action tests.
	RunAdminUpdate                    bool   // specify yes or no if Admin Update tests need to be run
	OSAClusterVersion                 string // OSA Cluster Version specifies the latest cluster version of RP
	LatestMasterEtcdImageVersion      string // Latest Master Etcd ImageVersion
	LatestPrometheusImageVersion      string // Latest Prometheus Image Version
	LatestEtcdBackupImageVersion      string // Latest Etcd Backup Image Version
	ValidatorBilling                  bool
	OSAE2EReuseCluster                string // Values: "DELETE" deletes the cluster and creates if it doesn't exist. Anything else creates the cluster if not exist and skips deleting it.
	OSAE2EReuseClusterUniqueId        string // This is used as a blob name in test storage in a common container. Ensure this is a valid azure blob name.
	OSABillingE2EClusterUniqueID      string // This is used as a blob name in test storage in a common container. It will be picked up by the billing e2es to get cluster information
	PrivateLinkFeature                bool   // This flag enabling Private Link feature to be deployed in ARO Cluster
}

var intv2EnabledSubs = []string{
	"8ecadfc9-d1a3-4ea4-b844-0d9f87e4d7c8",
	"5299e6b7-b23b-46c8-8277-dc1147807117",
}

// ValidateListByRegionStrategy represents the strategy used to perform admin action ListClustersByRegion
type ValidateListByRegionStrategy string

const (
	// NotValidate indicates do not validation
	NotValidate = ValidateListByRegionStrategy("none")
	// PartialValidate indicates do partial validation
	PartialValidate = ValidateListByRegionStrategy("partial")
	// FullValidate indicates do full validation
	FullValidate = ValidateListByRegionStrategy("full")
)

var armURL string // the actual url for the arm Endpoint
var armURLLock sync.Mutex

var aadURL string // the actual url for the aad Endpoint
var kvURL string  // the actual url for the Key Vault Endpoint

var location string

// ARMURL returns the ARM url at which to construct clients. See also ARMURLProd().
func ARMURL() string {
	if len(armURL) != 0 {
		return armURL
	}
	// we should protect the url not only because it's global static variable.
	// but also the testproxy.Init will start a server, we should not start it twice.
	armURLLock.Lock()
	defer armURLLock.Unlock()

	if len(armURL) == 0 {
		cloud, err := azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", instance.Cloud))
		if err != nil {
			_, err := url.Parse(instance.Cloud)
			if err != nil {
				log.Fatalf("Provided cloud %s is invalid; must be a valid Azure cloud ('public', 'china', 'german', 'usgovernment') or a valid URL", instance.Cloud)
			}
			armURL = instance.Cloud
		} else {
			armURL = cloud.ResourceManagerEndpoint
		}

	}

	return armURL
}

// ARMURLProd returns the ARM url at which to construct clients disregarding the custom dev ARM endpoint.
// Azure SDK Authorizer defaults to public ARM endpoint when it's not specified. Therefore, for multiple Azure Cloud
// support, we have to send the correct ARM endpoint depending on the cloud. Currently, we overload the ARM endpoint
// (see ARMURL()) which returns a custom URL which will not be accepted by the Authorizer.
// If cloud is set to custom value (for OSA), we will still default to public cloud.
func ARMURLProd() string {
	cloud, err := azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", instance.Cloud))

	if err != nil {
		log.Printf("Cloud set to %s, assuming public cloud for Authorizer ARM URL endpoint", instance.Cloud)
		cloud, err = azure.EnvironmentFromName("AZUREPUBLICCLOUD")
		if err != nil {
			log.Fatal("Unexpected error obtaining public cloud")
		}
	}

	return cloud.ResourceManagerEndpoint
}

// AADURL returns the AAD url at which to construct authorizers. See also ARMURLProd().
func AADURL() string {
	if len(aadURL) != 0 {
		return aadURL
	}

	cloud, err := azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", instance.Cloud))
	if err != nil {
		log.Printf("Cloud set to %s, assuming public cloud for Authorizer AAD URL endpoint", instance.Cloud)
		cloud, err = azure.EnvironmentFromName("AZUREPUBLICCLOUD")
		if err != nil {
			log.Fatal("Unexpected error obtaining public cloud")
		}
	}

	aadURL = cloud.ActiveDirectoryEndpoint

	return aadURL
}

// KVURL returns the Key Vault url at which to construct authorizers.
func KVURL() string {
	if len(kvURL) != 0 {
		return kvURL
	}

	cloud, err := azure.EnvironmentFromName(fmt.Sprintf("AZURE%sCLOUD", instance.Cloud))
	if err != nil {
		log.Printf("Cloud set to %s, assuming public cloud for Authorizer KV URL endpoint", instance.Cloud)
		cloud, err = azure.EnvironmentFromName("AZUREPUBLICCLOUD")
		if err != nil {
			log.Fatal("Unexpected error obtaining public cloud")
		}
	}

	kvURL = cloud.KeyVaultEndpoint

	return kvURL
}

// GetRandomRegion selects a location from the list of available regions
func GetRandomRegion() string {
	if len(location) != 0 {
		return location
	}

	if instance.Regions == nil || len(instance.Regions) == 0 {
		log.Fatalf("Regions should not be empty")
	}
	log.Printf("Picking random region from list %s\n", instance.Regions)
	location = instance.Regions[testrandom.RandomInteger(len(instance.Regions))]
	location = strings.ToLower(location)
	log.Printf("Picked region %s\n", location)

	return location
}

// GetTimeout returns EventuallyTimeout
func GetTimeout() time.Duration {
	return instance.EventuallyTimeout
}

// IsOSAClusterType returns true if the cluster type used is OSA
func IsOSAClusterType() bool {
	return strings.ToLower(instance.ClusterType) == "osa"
}

// GetClusterVMSize returns the vm size used for cluster
func GetClusterVMSize() string {
	if instance.ClusterVMSize != "" {
		return (instance.ClusterVMSize)
	}
	return string(containerservice.StandardDS1V2)
}

// GetOsDiskSizeGB returns the os disk size used for cluster nodes
func GetOsDiskSizeGB() int32 {
	if instance.OsDiskSizeGB == 0 {
		return 30
	}
	return instance.OsDiskSizeGB
}

// CreateNodeCount returns the number of nodes for the cluster create
func CreateNodeCount() int {
	return instance.ClusterNodeCount
}

// ScaleNodeCounts returns the number of nodes to scale down and scale up to
func ScaleNodeCounts() (min int, max int) {
	return instance.ScaleMinCount, instance.ScaleMaxCount
}

// StorageAccountTestName returns name for test storage account
func StorageAccountTestName() string {
	return instance.OSAConfig.StorageAccountNameClusterConfig
}

// StorageAccountTestKVSecret returns the keyvault uri for secret of test storage account
func StorageAccountTestKVSecret() string {
	return instance.OSAConfig.StorageAccountSecretClusterConfig
}

// OSADeepClusterTestsEnabled returns true if deep cluster tests are enabled
func OSADeepClusterTestsEnabled() bool {
	// Deactivating Deep tests in case of private cluster
	if GetPrivateLinkFeature() {
		return false
	}
	return instance.OSAConfig.EnableDeepTests
}

func DeletionDueTime() string {
	return instance.DeletionDueTime
}

// RPEndpoint returns the private ip of the RP
func RPEndpoint() string {
	return instance.RpEndpoint
}

// IsOSAPrivateRP returns true if the RP is running locally or on a private IP
func IsOSAPrivateRP() bool {
	return instance.RpEndpoint != ""
}

// IsOSAINTRP returns true if the RP is calling intv2 environment
func IsOSAINTRP(subscription, region string) bool {
	return IsIntRegion(region) && IsIntEnabledSub(subscription)
}

// IsIntRegion returns true if the region has an intv2 endpoint that can be called
func IsIntRegion(region string) bool {
	return (strings.EqualFold(region, "eastus") || strings.EqualFold(region, "westus2"))
}

// IsIntEnabledSub returns true if the subscription can hit intv2
func IsIntEnabledSub(subscription string) bool {
	for _, sub := range intv2EnabledSubs {
		if strings.EqualFold(sub, subscription) {
			return true
		}
	}
	return false
}

// isStoreageAccountTestPresent returns true if both Account name and KV uri is configured
func isStoreageAccountTestPresent() bool {
	return (instance.OSAConfig.StorageAccountNameClusterConfig != "" && instance.OSAConfig.StorageAccountSecretClusterConfig != "")
}

// GetOSAKVURI returns the regional OSA KV used to generated TLS CA Signed certificates
func GetOSAKVURI() string {
	return instance.OSAConfig.OSAKV
}

// RunValidatorOSATLS returns true if the OSA TLS Validator has to be run
func RunValidatorOSATLS() bool {
	return instance.OSAConfig.ValidatorOSATLS
}

// RunValidatorBilling returns true if the Billing Validator has to be run
func RunValidatorBilling() bool {
	return instance.OSAConfig.ValidatorBilling
}

// InternalSubscriptions represents internal subscriptions
var InternalSubscriptions = []string{"8ecadfc9-d1a3-4ea4-b844-0d9f87e4d7c8"}

// GetValidateListByRegionStrategy returns strategy to run ListClustersByRegion admin action
func GetValidateListByRegionStrategy(subscription, region string) ValidateListByRegionStrategy {
	strategy := ValidateListByRegionStrategy(strings.ToLower(strings.TrimSpace(instance.OSAConfig.ValidateListByRegionStrategy)))

	// change fullvalidate to partialvalidate if it's not an internal sub and not from eastus, doing so to avoid full validate as it has a higher operation cost
	if strategy == FullValidate {
		strategy = PartialValidate
		for _, internalSub := range InternalSubscriptions {
			if subscription == internalSub && region == "eastus" {
				strategy = FullValidate
				break
			}
		}
	}
	return strategy
}

// GetOSAAdminKVURI returns the OSA Admin KV used to access certificates for admin operations
func GetOSAAdminKVURI() string {
	return instance.OSAConfig.OSAAdminKV
}

// RunAdminUpdate returns true if the OSA Run Admin Update has to be run
func RunAdminUpdate() bool {
	return instance.OSAConfig.RunAdminUpdate
}

// RunAdminGenevaActionAsyncTest returns true if the OSA long run Admin Geneva Action test has to be run
func RunAdminGenevaActionAsyncTest() bool {
	return instance.OSAConfig.EnableAdminAsyncGenevaActionTest
}

// RunValidatorOSAPluginTLS returns true if the OSA TLS Plugin Validator has to be run
func RunValidatorOSAPluginTLS() bool {
	return instance.OSAConfig.ValidatorOSAPluginTLS
}

// GetRPCredsAppID return the OSA RP Creds App ID used to read cluster KV
func GetRPCredsAppID() string {
	return instance.OSAConfig.RPCredsAppID
}

// GetRPSecretPath return the OSA RP Creds Secret Path used to read cluster KV
func GetRPSecretPath() string {
	return instance.OSAConfig.RPCredsSecretPath
}

// GetOSAPublisherTenantID return the OSA Publisher Tenant ID
func GetOSAPublisherTenantID() string {
	return instance.OSAConfig.OSAPublisherTenantID
}

// GetOSAClusterVersion return the OSA Cluster Version
func GetOSAClusterVersion() string {
	return instance.OSAConfig.OSAClusterVersion
}

// GetOSAAdminEndpoint return the OSA Admin Endpoint
func GetOSAAdminEndpoint() string {
	return instance.OSAConfig.OSAAdminEndpoint
}

// GetLatestMasterEtcdImageVersion return the OSA LatestMasterEtcdImageVersion
func GetLatestMasterEtcdImageVersion() string {
	return instance.OSAConfig.LatestMasterEtcdImageVersion
}

// GetLatestPrometheusImageVersion return the OSA LatestPrometheusImageVersion
func GetLatestPrometheusImageVersion() string {
	return instance.OSAConfig.LatestPrometheusImageVersion
}

// GetLatestEtcdBackupImageVersion return the OSA LatestEtcdBackupImageVersion
func GetLatestEtcdBackupImageVersion() string {
	return instance.OSAConfig.LatestEtcdBackupImageVersion
}

// GetOSAE2EReuseCluster return the OSA reuse cluster configuration
func GetOSAE2EReuseCluster() string {
	return instance.OSAConfig.OSAE2EReuseCluster
}

// GetOSAE2EReuseClusterUniqueId return the OSA reuse cluster configuration
func GetOSAE2EReuseClusterUniqueId() string {
	return instance.OSAConfig.OSAE2EReuseClusterUniqueId
}

// GetOSABillingE2EClusterUniqueID return the OSA billing e2e cluster configuration
func GetOSABillingE2EClusterUniqueID() string {
	return instance.OSAConfig.OSABillingE2EClusterUniqueID
}

// GetPrivateLinkFeature return if we want either or not a private link feature enable cluster - forcing true for WCUS and eastus2
func GetPrivateLinkFeature() bool {
	if isPrivateLinkRegion(location) && strings.Contains(instance.SuiteName, "vnet") {
		instance.OSAConfig.PrivateLinkFeature = true
	}
	return instance.OSAConfig.PrivateLinkFeature
}

func isPrivateLinkRegion(location string) bool {
	return strings.EqualFold(location, "eastus2") ||
		strings.EqualFold(location, "westcentralus") ||
		strings.EqualFold(location, "northeurope") ||
		strings.EqualFold(location, "southafricanorth") ||
		strings.EqualFold(location, "northcentralus")
}

// SetSuiteName save the suite name running for future validation
func SetSuiteName(suiteName string) {
	instance.SuiteName = suiteName
}

// Log prints the config in logs
func Log() {
	log.Printf("Suite Name Test: %s", instance.SuiteName)
	log.Println()
	log.Printf("ARM Endpoint Base URI: %s", ARMURL())
	if IsOSAPrivateRP() {
		log.Printf("Using OSA Private RP %s", instance.RpEndpoint)
	}
	log.Println()
	log.Printf("Regions: %s", instance.Regions)
	log.Printf("DeletionDueTime: %s", instance.DeletionDueTime)
	log.Println()
	log.Printf("Eventually Timeout: %s", instance.EventuallyTimeout)
	log.Printf("Eventually Interval: %s", instance.EventuallyInterval)
	log.Println()
	log.Printf("Clean-up condition: %s", instance.CleanUpCondition)
	log.Println()
	log.Printf("Cluster VM Size: %s", instance.ClusterVMSize)
	log.Printf("Cluster Node Count: %d", instance.ClusterNodeCount)
	log.Printf("Scale min nodes: %d", instance.ScaleMinCount)
	log.Printf("Scale max nodes: %d", instance.ScaleMaxCount)
	log.Println()
	if isStoreageAccountTestPresent() {
		log.Printf("Using Test Storage account : %s", instance.OSAConfig.StorageAccountNameClusterConfig)
		log.Printf("Using Test Storage secret : %s", instance.OSAConfig.StorageAccountSecretClusterConfig)
	}
	log.Println()
	log.Printf("OSA Key Vault URI : %s", instance.OSAConfig.OSAKV)
	log.Printf("OSA RP Creds APP ID : %s", instance.OSAConfig.RPCredsAppID)
	log.Printf("OSA RP Creds Secret URI : %s", instance.OSAConfig.RPCredsSecretPath)
	log.Printf("OSA Publisher Tenant ID : %s", instance.OSAConfig.OSAPublisherTenantID)
	log.Printf("OSAE2EReuseClusterUniqueId set to : %s", GetOSAE2EReuseClusterUniqueId())
	log.Printf("OSABillingE2EClusterUniqueID set to : %s", GetOSABillingE2EClusterUniqueID())
	log.Println()
	log.Printf("Validator TLS Scenario set to : %t", RunValidatorOSATLS())
	log.Printf("Validator TLS Plugin Scenario set to : %t", RunValidatorOSAPluginTLS())
	log.Println()
	log.Printf("Deep test run set to : %t", OSADeepClusterTestsEnabled())
	log.Println()
	log.Printf("Admin Update tests set to : %t", RunAdminUpdate())
	log.Println()
	log.Printf("OSAE2EReuseCluster set to : %s", GetOSAE2EReuseCluster())
	log.Printf("ValidateListByRegionStrategy: %s", instance.OSAConfig.ValidateListByRegionStrategy)
	log.Println()
	log.Printf("Private Cluster set to : %t", GetPrivateLinkFeature())
	log.Println()
}

// AdminEndpoint returns the admin endpoint
func AdminEndpoint(region string) string {
	return instance.AdminEndpointFormat
}
