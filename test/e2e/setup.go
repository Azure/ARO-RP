package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/davecgh/go-spew/spew"
	"github.com/jongio/azidext/go/azidext"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	redhatopenshift20231122 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2023-11-22/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

var (
	disallowedInFilenameRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	DefaultEventuallyTimeout  = 5 * time.Minute
)

type clientSet struct {
	Operations        redhatopenshift20231122.OperationsClient
	OpenshiftClusters redhatopenshift20231122.OpenShiftClustersClient

	VirtualMachines       compute.VirtualMachinesClient
	Resources             features.ResourcesClient
	VirtualNetworks       network.VirtualNetworksClient
	DiskEncryptionSets    compute.DiskEncryptionSetsClient
	Disks                 compute.DisksClient
	NetworkSecurityGroups network.SecurityGroupsClient
	Subnet                network.SubnetsClient
	Interfaces            network.InterfacesClient
	Storage               storage.AccountsClient
	LoadBalancers         network.LoadBalancersClient

	RestConfig         *rest.Config
	HiveRestConfig     *rest.Config
	Monitoring         monitoringclient.Interface
	Kubernetes         kubernetes.Interface
	Client             client.Client
	MachineAPI         machineclient.Interface
	MachineConfig      mcoclient.Interface
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
	osClusterVersion  string
	clusterResourceID string
	clients           *clientSet
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

func skipIfNotHiveManagedCluster(adminAPICluster *admin.OpenShiftCluster) {
	if adminAPICluster.Properties.HiveProfile == (admin.HiveProfile{}) {
		Skip("skipping tests because this ARO cluster has not been created/adopted by Hive")
	}
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
	hubAddress := os.Getenv("ARO_SELENIUM_HOSTNAME")
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
		log.Infof("Could not get to %s. With error : %s", host, err.Error())
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

	configv1, err := kubeadminkubeconfig.Get(ctx, log, _env, authorizer, resourceIDFromEnv())
	if err != nil {
		return nil, err
	}

	var config api.Config
	err = latest.Scheme.Convert(configv1, &config, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(config, &clientcmd.ConfigOverrides{})

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	controllerRuntimeClient, err := client.New(restconfig, client.Options{})
	if err != nil {
		return nil, err
	}

	monitoring, err := monitoringclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	machineapicli, err := machineclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	mcocli, err := mcoclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	projectcli, err := projectclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	configcli, err := configclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	securitycli, err := securityclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	var hiveRestConfig *rest.Config
	var hiveClientSet client.Client
	var hiveAKS *kubernetes.Clientset
	var hiveCM hive.ClusterManager

	if _env.IsLocalDevelopmentMode() {
		liveCfg, err := _env.NewLiveConfigManager(ctx)
		if err != nil {
			return nil, err
		}

		hiveShard := 1
		hiveRestConfig, err = liveCfg.HiveRestConfig(ctx, hiveShard)
		if err != nil {
			return nil, err
		}

		hiveClientSet, err = client.New(hiveRestConfig, client.Options{})
		if err != nil {
			return nil, err
		}

		hiveAKS, err = kubernetes.NewForConfig(hiveRestConfig)
		if err != nil {
			return nil, err
		}

		hiveCM, err = hive.NewFromConfig(log, _env, hiveRestConfig)
		if err != nil {
			return nil, err
		}
	}

	return &clientSet{
		Operations:        redhatopenshift20231122.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClusters: redhatopenshift20231122.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		VirtualMachines:       compute.NewVirtualMachinesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Resources:             features.NewResourcesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		VirtualNetworks:       network.NewVirtualNetworksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Disks:                 compute.NewDisksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		DiskEncryptionSets:    compute.NewDiskEncryptionSetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Subnet:                network.NewSubnetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Interfaces:            network.NewInterfacesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		NetworkSecurityGroups: network.NewSecurityGroupsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Storage:               storage.NewAccountsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		LoadBalancers:         network.NewLoadBalancersClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		RestConfig:         restconfig,
		HiveRestConfig:     hiveRestConfig,
		Kubernetes:         cli,
		Client:             controllerRuntimeClient,
		Monitoring:         monitoring,
		MachineAPI:         machineapicli,
		MachineConfig:      mcocli,
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
	err := env.ValidateVars(
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"CLUSTER",
		"LOCATION")

	if err != nil {
		return err
	}

	_env, err = env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

	vnetResourceGroup = os.Getenv("RESOURCEGROUP") // TODO: remove this when we deploy and peer a vnet per cluster create
	if os.Getenv("CI") != "" {
		vnetResourceGroup = os.Getenv("CLUSTER")
	}
	clusterName = os.Getenv("CLUSTER")

	osClusterVersion = os.Getenv("OS_CLUSTER_VERSION")

	if os.Getenv("CI") != "" { // always create cluster in CI
		cluster, err := cluster.New(log, _env, os.Getenv("CI") != "")
		if err != nil {
			return err
		}

		if osClusterVersion == "" {
			osClusterVersion = version.DefaultInstallStream.Version.String()
		}

		err = cluster.Create(ctx, vnetResourceGroup, clusterName, osClusterVersion)
		if err != nil {
			return err
		}
	}

	clusterResourceID = resourceIDFromEnv()

	clients, err = newClientSet(ctx)
	if err != nil {
		return err
	}

	return nil
}

func done(ctx context.Context) error {
	// terminate early if delete flag is set to false
	if os.Getenv("CI") != "" && os.Getenv("E2E_DELETE_CLUSTER") != "false" {
		cluster, err := cluster.New(log, _env, os.Getenv("CI") != "")
		if err != nil {
			return err
		}

		err = cluster.Delete(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			return err
		}
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
