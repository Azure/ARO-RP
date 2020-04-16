package testresources

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/zhuoli/ARO-RP/test/e2e/helpers/testconfig"
	"github.com/zhuoli/ARO-RP/test/e2e/objects/azure"
)

// ResourceManagerInstance is the singleton instance for resource management
type ResourceManagerInstance struct {
	creds *auth.ClientCredentialsConfig

	registeredRGs map[string][]*azure.ResourceGroup
}

// AddClusterBlobForBillingE2E tries to insert a blob containing information about the cluster which can be picked up by th billing e2e tests
func AddClusterBlobForBillingE2E(subscriptionID, resourceGroupName, resourceName string) error {
	// container pre-exists by name i.e "<region>arobillinge2ecluster"
	containerName := fmt.Sprintf("%sarobillinge2ecluster", testconfig.GetRandomRegion())
	r := resourceManager()
	if testconfig.StorageAccountTestName() == "" {
		return fmt.Errorf("Test configuration doesn't have storageAccountTestName configured")
	}

	if testconfig.StorageAccountTestKVSecret() == "" {
		return fmt.Errorf("Test configuration doesn't have storageAccountTestKVSecret configured")
	}

	storageSecret, err := r.clientSet.Secret().Get(testconfig.StorageAccountTestKVSecret())
	if err != nil {
		return fmt.Errorf("Failed to get storage key from keyvault :%s", err)
	}

	blobName := fmt.Sprintf("%s_%s_%s", testconfig.GetOSABillingE2EClusterUniqueID(), testconfig.GetRandomRegion(), subscriptionID)
	log.Printf("AddClusterBlobForBillingE2E: trying to save cluster information to blob: %s in container: %s", blobName, containerName)

	var clusterInfo = TestClusterState{
		ClusterName:              resourceName,
		ManagedResourceGroupName: resourceGroupName,
	}

	blobContent, err := json.Marshal(clusterInfo)
	if err != nil {
		log.Printf("failed to marshal clusterInfo into bytes, err: %v", err)
		return fmt.Errorf("failed to marshal clusterInfo into bytes, err: %v", err)
	}

	err = r.clientSet.Storage().UploadBlob(
		testconfig.StorageAccountTestName(),
		*storageSecret.Value(),
		containerName,
		blobName,
		blobContent)
	if err != nil {
		return fmt.Errorf("Failed to upload blob to storage :%s", err)
	}

	log.Printf("AddClusterBlobForBillingE2E: successfully saved cluster information: %+v to blob: %s in container: %s", clusterInfo, blobName, containerName)
	return nil
}
