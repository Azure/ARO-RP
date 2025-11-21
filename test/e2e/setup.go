package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/davecgh/go-spew/spew"
	"github.com/jongio/azidext/go/azidext"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/azure"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/common"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	redhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	utilcluster "github.com/Azure/ARO-RP/pkg/util/cluster"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/test/util/dynamic"
)

const (
	smoke = "smoke"
	// regressiontest is for tests designed to ensure that something doesn't
	// break before we go to release, but doesn't need to be validated in every
	// PR.
	regressiontest = "regressiontest"
)

//go:embed static_resources
var staticResources embed.FS

var (
	disallowedInFilenameRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	DefaultEventuallyTimeout  = 5 * time.Minute
)

type clientSet struct {
	Operations        redhatopenshift20240812preview.OperationsClient
	OpenshiftClusters redhatopenshift20240812preview.OpenShiftClustersClient

	VirtualMachines       compute.VirtualMachinesClient
	Resources             features.ResourcesClient
	DiskEncryptionSets    compute.DiskEncryptionSetsClient
	Disks                 compute.DisksClient
	Interfaces            armnetwork.InterfacesClient
	LoadBalancers         armnetwork.LoadBalancersClient
	NetworkSecurityGroups armnetwork.SecurityGroupsClient
	Subnet                armnetwork.SubnetsClient
	VirtualNetworks       armnetwork.VirtualNetworksClient
	Storage               storage.AccountsClient

	Dynamic            dynamic.Client
	RestConfig         *rest.Config
	HiveRestConfig     *rest.Config
	Monitoring         monitoringclient.Interface
	Kubernetes         kubernetes.Interface
	Client             client.Client
	MachineAPI         machineclient.Interface
	MachineConfig      mcoclient.Interface
	Route              routeclient.Interface
	AROClusters        aroclient.Interface
	ConfigClient       configclient.Interface
	SecurityClient     securityclient.Interface
	Project            projectclient.Interface
	Hive               client.Client
	HiveAKS            kubernetes.Interface
	HiveClusterManager hive.ClusterManager
}

var (
	log               *logrus.Entry
	_env              env.Core
	vnetResourceGroup string
	clusterName       string
	clusterResourceID string
	clients           *clientSet
	isMiwi            bool
)

func skipIfNotInDevelopmentEnv() {
	if !_env.IsLocalDevelopmentMode() {
		Skip("skipping tests in non-development environment")
	}
}

func skipIfSeleniumNotEnabled() {
	if os.Getenv("ARO_SELENIUM_HOSTNAME") == "" {
		Skip("ARO_SELENIUM_HOSTNAME not set, skipping portal e2e")
	}
}

func skipIfMIMOActuatorNotEnabled() {
	if os.Getenv("ARO_E2E_MIMO") == "" {
		Skip("ARO_E2E_MIMO not set, skipping MIMO e2e")
	}
}

func skipIfNotHiveManagedCluster(adminAPICluster *admin.OpenShiftCluster) {
	if !isHiveManagedCluster(adminAPICluster) {
		Skip("skipping tests because this ARO cluster has not been created/adopted by Hive")
	}
}

func isHiveManagedCluster(adminAPICluster *admin.OpenShiftCluster) bool {
	return adminAPICluster.Properties.HiveProfile != (admin.HiveProfile{})
}

func SaveScreenshot(wd selenium.WebDriver, e error) {
	log.Infof("Error : %s", e.Error())
	log.Info("Taking Screenshot and saving page source")
	imageBytes, err := wd.Screenshot()
	if err != nil {
		panic(err)
	}

	sourceString, err := wd.PageSource()
	if err != nil {
		panic(err)
	}

	errorString := disallowedInFilenameRegex.ReplaceAllString(e.Error(), "_")

	// If the string is too long, snip it and add a random component, keeping to
	// 100 characters total filename length once the file type is added on
	if len(errorString) > 95 {
		errorString = errorString[:59] + "_" + uuid.DefaultGenerator.Generate()
	}

	imagePath := "./" + errorString + ".png"
	sourcePath := "./" + errorString + ".html"

	imageAbsPath, err := filepath.Abs(imagePath)
	if err != nil {
		panic(err)
	}
	sourceAbsPath, err := filepath.Abs(sourcePath)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(imageAbsPath, imageBytes, 0666)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(sourceAbsPath, []byte(sourceString), 0666)
	if err != nil {
		panic(err)
	}

	log.Infof("Screenshot saved to %s", imageAbsPath)
	log.Infof("Page Source saved to %s", sourceAbsPath)
}

func adminPortalSessionSetup() (string, *selenium.WebDriver) {
	const (
		hubPort  = 4444
		hostPort = 8444
	)
	hubAddress, exists := os.LookupEnv("ARO_SELENIUM_HOSTNAME")
	if !exists {
		hubAddress = "localhost"
	}
	os.Setenv("SE_SESSION_REQUEST_TIMEOUT", "9000")

	caps := selenium.Capabilities{
		"browserName":         "MicrosoftEdge",
		"acceptInsecureCerts": true,
	}
	wd := selenium.WebDriver(nil)

	_, err := url.ParseRequestURI(fmt.Sprintf("https://%s:%d", hubAddress, hubPort))
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		wd, err = selenium.NewRemote(caps, fmt.Sprintf("http://%s:%d/wd/hub", hubAddress, hubPort))
		if wd != nil {
			err = nil
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		panic(err)
	}

	log := utillog.GetLogger()

	// Navigate to the simple playground interface.
	host, exists := os.LookupEnv("PORTAL_HOSTNAME")
	if !exists {
		host = fmt.Sprintf("https://localhost:%d", hostPort)
	}

	if err := wd.Get(host + "/healthz/ready"); err != nil {
		log.Infof("Could not get to %s. With error : %s", host+"/healthz/ready", err.Error())
	}

	mainPortalPath := host + "/portal"
	if err := wd.Get(mainPortalPath); err != nil {
		log.Infof("Failed to reach main portal path at %s. Error: %s", mainPortalPath, err.Error())
	}
	var portalAuthCmd string
	var portalAuthArgs = make([]string, 0)
	if os.Getenv("CI") != "" {
		// In CI we have a prebuilt portalauth binary
		portalAuthCmd = "./portalauth"
	} else {
		portalAuthCmd = "go"
		portalAuthArgs = []string{"run", "./hack/portalauth"}
	}

	portalAuthArgs = append(portalAuthArgs, "-username", "test", "-groups", "$AZURE_PORTAL_ELEVATED_GROUP_IDS")

	cmd := exec.Command(portalAuthCmd, portalAuthArgs...)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error occurred creating session cookie\n Output: %s\n Error: %s\n", output, err)
	}

	os.Setenv("SESSION", string(output))

	log.Infof("Session Output : %s\n", os.Getenv("SESSION"))

	cookie := &selenium.Cookie{
		Name:   "session",
		Value:  os.Getenv("SESSION"),
		Expiry: math.MaxUint32,
	}

	if err := wd.AddCookie(cookie); err != nil {
		panic(err)
	}
	return host, &wd
}

func resourceIDFromEnv() string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s",
		_env.SubscriptionID(), vnetResourceGroup, clusterName)
}

func newClientSet(ctx context.Context) (*clientSet, error) {
	options := _env.Environment().EnvironmentCredentialOptions()
	tokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		return nil, err
	}

	scopes := []string{_env.Environment().ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)

	res, err := azure.ParseResourceID(resourceIDFromEnv())
	if err != nil {
		return nil, err
	}

	clusters := redhatopenshift20240812preview.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer)

	r, err := clusters.ListAdminCredentials(ctx, res.ResourceGroup, res.ResourceName)
	if err != nil {
		return nil, err
	}

	kubeConfigFile, err := base64.StdEncoding.DecodeString(*r.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error b64 decoding kubeconfig file: %w", err)
	}

	kubeconfig, err := clientcmd.NewClientConfigFromBytes(kubeConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error building clientconfig from bytes: %w", err)
	}

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error building clientconfig: %w", err)
	}

	// In development e2e the certificate is not always created with a publicly
	// verifiable TLS certificate.
	if _env.IsLocalDevelopmentMode() {
		restconfig.Insecure = true // CodeQL [SM03511] only used in local development
	} else {
		// In prod e2e there is sometimes a lag between cluster creation and the
		// TLS certificate being presented by the APIServer
		configShallowCopy := *restconfig

		// Create a HTTPClient which we can
		httpClient, err := rest.HTTPClientFor(&configShallowCopy)
		if err != nil {
			return nil, err
		}
		configShallowCopy.GroupVersion = &schema.GroupVersion{}
		configShallowCopy.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
		rawClient, err := rest.RESTClientForConfigAndClient(&configShallowCopy, httpClient)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		timeoutDuration := 20 * time.Minute
		sleepAmount := 10 * time.Second
		maxRetryCount := int(timeoutDuration) / int(sleepAmount)
		passed := false

		for i := range maxRetryCount {
			var statusCode int
			err = rawClient.
				Get().
				AbsPath("/healthz").
				Do(ctx).
				StatusCode(&statusCode).
				Error()

			if err != nil {
				log.Warnf("API Server not ready (try %d/%d): %s", i+1, maxRetryCount, err.Error())
			} else if statusCode != 200 {
				log.Warnf("API Server not ready (try %d/%d): status code %d", i+1, maxRetryCount, statusCode)
			} else {
				passed = true
				break
			}

			time.Sleep(sleepAmount)
		}

		if passed {
			log.Infof("API Server ready after %s", time.Since(now).String())
		} else {
			return nil, fmt.Errorf("timed out waiting for API server to be ready after %s", timeoutDuration.String())
		}
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubernetes clientset: %w", err)
	}

	controllerRuntimeClient, err := client.New(restconfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("error building controller runtime client: %w", err)
	}

	monitoring, err := monitoringclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building monitoring client: %w", err)
	}

	machineapicli, err := machineclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building machine API client: %w", err)
	}

	mcocli, err := mcoclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building MCO client: %w", err)
	}

	projectcli, err := projectclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building project client: %w", err)
	}

	routecli, err := routeclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building route client: %w", err)
	}

	arocli, err := aroclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building ARO k8s client: %w", err)
	}

	configcli, err := configclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building config client: %w", err)
	}

	securitycli, err := securityclient.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building security client: %w", err)
	}

	dynamiccli, err := dynamic.NewDynamicClient(restconfig)
	if err != nil {
		return nil, fmt.Errorf("error building dynamic k8s client: %w", err)
	}

	var hiveRestConfig *rest.Config
	var hiveClientSet client.Client
	var hiveAKS *kubernetes.Clientset
	var hiveCM hive.ClusterManager

	liveCfg, err := _env.NewLiveConfigManager(ctx)
	if err != nil {
		return nil, err
	}

	adoptByHive, err := liveCfg.AdoptByHive(ctx)
	if err != nil {
		return nil, err
	}

	installViaHive, err := liveCfg.InstallViaHive(ctx)
	if err != nil {
		return nil, err
	}

	if _env.IsLocalDevelopmentMode() && (adoptByHive || installViaHive) {
		hiveShard := 1
		hiveRestConfig, err = liveCfg.HiveRestConfig(ctx, hiveShard)
		if err != nil {
			return nil, fmt.Errorf("error getting hive RESTConfig: %w", err)
		}

		hiveClientSet, err = client.New(hiveRestConfig, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("error building Hive client: %w", err)
		}

		hiveAKS, err = kubernetes.NewForConfig(hiveRestConfig)
		if err != nil {
			return nil, fmt.Errorf("error building Hive AKS client: %w", err)
		}

		hiveCM, err = hive.NewFromConfigClusterManager(log, _env, hiveRestConfig)
		if err != nil {
			return nil, fmt.Errorf("error building Hive cluster manager: %w", err)
		}
	}

	clientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud:           _env.Environment().Cloud,
			Retry:           common.RetryOptions,
			PerCallPolicies: []policy.Policy{azureclient.NewLoggingPolicy()},
		},
	}

	interfacesClient, err := armnetwork.NewInterfacesClient(_env.SubscriptionID(), tokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	loadBalancersClient, err := armnetwork.NewLoadBalancersClient(_env.SubscriptionID(), tokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := armnetwork.NewSecurityGroupsClient(_env.SubscriptionID(), tokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	subnetsClient, err := armnetwork.NewSubnetsClient(_env.SubscriptionID(), tokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	virtualNetworksClient, err := armnetwork.NewVirtualNetworksClient(_env.SubscriptionID(), tokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		Operations:        redhatopenshift20240812preview.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClusters: clusters,

		VirtualMachines:       compute.NewVirtualMachinesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Resources:             features.NewResourcesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Disks:                 compute.NewDisksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		DiskEncryptionSets:    compute.NewDiskEncryptionSetsClientWithAROEnvironment(_env.Environment(), _env.SubscriptionID(), authorizer),
		Interfaces:            interfacesClient,
		LoadBalancers:         loadBalancersClient,
		NetworkSecurityGroups: securityGroupsClient,
		Subnet:                subnetsClient,
		VirtualNetworks:       virtualNetworksClient,
		Storage:               storage.NewAccountsClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		RestConfig:         restconfig,
		HiveRestConfig:     hiveRestConfig,
		Kubernetes:         cli,
		Dynamic:            dynamiccli,
		Client:             controllerRuntimeClient,
		Monitoring:         monitoring,
		MachineAPI:         machineapicli,
		MachineConfig:      mcocli,
		Route:              routecli,
		AROClusters:        arocli,
		Project:            projectcli,
		ConfigClient:       configcli,
		SecurityClient:     securitycli,
		Hive:               hiveClientSet,
		HiveAKS:            hiveAKS,
		HiveClusterManager: hiveCM,
	}, nil
}

func setup(ctx context.Context) error {
	if err := env.ValidateVars(
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"CLUSTER",
		"LOCATION"); err != nil {
		return err
	}

	// Core ARO env
	var err error
	_env, err = env.NewCoreForCI(ctx, log, env.SERVICE_E2E)
	if err != nil {
		return err
	}

	// Read out your test config
	conf, err := utilcluster.NewClusterConfigFromEnv()
	if err != nil {
		return err
	}

	// Build a bare‐bones Azure SDK client for OpenshiftClusters
	credOptions := _env.Environment().EnvironmentCredentialOptions()
	tokenCred, err := azidentity.NewEnvironmentCredential(credOptions)
	if err != nil {
		return err
	}
	scopes := []string{_env.Environment().ResourceManagerScope}
	authAdapter := azidext.NewTokenCredentialAdapter(tokenCred, scopes)
	azOCClient := redhatopenshift20240812preview.NewOpenShiftClustersClient(
		_env.Environment(), _env.SubscriptionID(), authAdapter)

	// Only check for leftover clusters in local dev CI, not in release E2E
	if conf.IsLocalDevelopmentMode() && conf.IsCI {
		const (
			maxRetries  = 10
			waitBetween = 30 * time.Second
		)
		totalWait := time.Duration(maxRetries) * waitBetween

		for attempt := 1; attempt <= maxRetries; attempt++ {
			doc, err := azOCClient.Get(ctx, conf.VnetResourceGroup, conf.ClusterName)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					log.Infof("No leftover cluster found on attempt %d; proceeding", attempt)
					break
				}
				return fmt.Errorf("failed to check leftover cluster (attempt %d): %w", attempt, err)
			}

			if doc.ProvisioningState != mgmtredhatopenshift20240812preview.Deleting {
				return fmt.Errorf("unexpected state %s on attempt %d; aborting", doc.ProvisioningState, attempt)
			}

			if attempt == maxRetries {
				return fmt.Errorf("cluster still stuck in Deleting after %s; aborting", totalWait)
			}

			log.Infof("Cluster still deleting (%d/%d); retrying in %s", attempt, maxRetries, waitBetween)
			time.Sleep(waitBetween)
		}
		// Old cluster is gone, create the new one
	}

	// we only create a cluster when running this in CI
	if conf.IsCI {
		cluster, err := utilcluster.New(log, conf)
		if err != nil {
			return err
		}
		if err = cluster.Create(ctx); err != nil {
			return err
		}
	}

	vnetResourceGroup = conf.VnetResourceGroup
	clusterName = conf.ClusterName
	clusterResourceID = resourceIDFromEnv()
	isMiwi = conf.UseWorkloadIdentity

	clients, err = newClientSet(ctx)
	if err != nil {
		return err
	}

	return nil
}

func done(ctx context.Context) error {
	// Load the usual cluster config (to pick up IsCI, etc.)
	conf, err := utilcluster.NewClusterConfigFromEnv()
	if err != nil {
		return err
	}

	if conf.IsCI && os.Getenv("E2E_DELETE_CLUSTER") != "false" {
		cluster, err := utilcluster.New(log, conf)
		if err != nil {
			return err
		}

		// Attempt deletion
		err = cluster.Delete(ctx, conf.VnetResourceGroup, conf.ClusterName)
		if err != nil {
			log.Errorf("Cluster deletion failed with errors: %v", err)
			return err
		}
		log.Info("Cluster deletion completed successfully")
	}

	return nil
}

var _ = BeforeSuite(func() {
	log.Info("BeforeSuite")

	SetDefaultEventuallyTimeout(DefaultEventuallyTimeout)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	if err := setup(context.Background()); err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			spew.Dump(oDataError.GetErrorEscaped())
		}
		panic(err)
	}
})

var _ = AfterSuite(func() {
	log.Info("AfterSuite")

	if err := done(context.Background()); err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			spew.Dump(oDataError.GetErrorEscaped())
		}
		panic(err)
	}
})
