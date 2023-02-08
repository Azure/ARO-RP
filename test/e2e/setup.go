package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	redhatopenshift20220904 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-09-04/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

const seleniumContainerName = "selenium-edge-standalone"

type clientSet struct {
	Operations        redhatopenshift20220904.OperationsClient
	OpenshiftClusters redhatopenshift20220904.OpenShiftClustersClient

	VirtualMachines       compute.VirtualMachinesClient
	Resources             features.ResourcesClient
	VirtualNetworks       network.VirtualNetworksClient
	DiskEncryptionSets    compute.DiskEncryptionSetsClient
	Disks                 compute.DisksClient
	NetworkSecurityGroups network.SecurityGroupsClient
	Subnet                network.SubnetsClient

	RestConfig         *rest.Config
	HiveRestConfig     *rest.Config
	Kubernetes         kubernetes.Interface
	MachineAPI         machineclient.Interface
	MachineConfig      mcoclient.Interface
	AROClusters        aroclient.Interface
	ConfigClient       configclient.Interface
	Project            projectclient.Interface
	Hive               hiveclient.Interface
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

	dockerSucceeded bool
)

func skipIfNotInDevelopmentEnv() {
	if !_env.IsLocalDevelopmentMode() {
		Skip("skipping tests in non-development environment")
	}
}

func skipIfDockerNotWorking() {
	// docker cmds will fail in INT until we figure out a solution since
	// it is running from docker already
	if !dockerSucceeded {
		Skip("skipping admin portal tests as docker is not available")
	}
}

func skipIfNotHiveManagedCluster(adminAPICluster *admin.OpenShiftCluster) {
	if adminAPICluster.Properties.HiveProfile == (admin.HiveProfile{}) {
		Skip("skipping tests because this ARO cluster has not been created/adopted by Hive")
	}
}

func SaveScreenshotAndExit(wd selenium.WebDriver, e error) {
	log.Infof("Error : %s", e.Error())
	log.Info("Taking Screenshot and saving page source")
	imageBytes, err := wd.Screenshot()
	if err != nil {
		panic(err)
	}

	imageData, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		panic(err)
	}

	sourceString, err := wd.PageSource()
	if err != nil {
		panic(err)
	}

	errorString := strings.ReplaceAll(e.Error(), " ", "_")

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

	image, err := os.Create(imageAbsPath)
	if err != nil {
		panic(err)
	}

	source, err := os.Create(sourceAbsPath)
	if err != nil {
		panic(err)
	}

	err = png.Encode(image, imageData)
	if err != nil {
		panic(err)
	}

	_, err = source.WriteString(sourceString)
	if err != nil {
		panic(err)
	}

	err = image.Close()
	if err != nil {
		panic(err)
	}

	err = source.Close()
	if err != nil {
		panic(err)
	}

	log.Infof("Screenshot saved to %s", imageAbsPath)
	log.Infof("Page Source saved to %s", sourceAbsPath)

	panic(e)
}

func adminPortalSessionSetup() (string, *selenium.WebDriver) {
	const (
		hubPort  = 4444
		hostPort = 8444
	)
	hubAddress := "localhost"
	if os.Getenv("AGENT_NAME") != "" {
		hubAddress = "selenium"
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

	if err := wd.Get(host + "/api/info"); err != nil {
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
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

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

	var hiveRestConfig *rest.Config
	var hiveClientSet *hiveclient.Clientset
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

		hiveClientSet, err = hiveclient.NewForConfig(hiveRestConfig)
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
		Operations:        redhatopenshift20220904.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClusters: redhatopenshift20220904.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		VirtualMachines:       compute.NewVirtualMachinesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Resources:             features.NewResourcesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		VirtualNetworks:       network.NewVirtualNetworksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Disks:                 compute.NewDisksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		DiskEncryptionSets:    compute.NewDiskEncryptionSetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Subnet:                network.NewSubnetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		NetworkSecurityGroups: network.NewSecurityGroupsClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		RestConfig:         restconfig,
		HiveRestConfig:     hiveRestConfig,
		Kubernetes:         cli,
		MachineAPI:         machineapicli,
		MachineConfig:      mcocli,
		AROClusters:        arocli,
		Project:            projectcli,
		ConfigClient:       configcli,
		Hive:               hiveClientSet,
		HiveAKS:            hiveAKS,
		HiveClusterManager: hiveCM,
	}, nil
}

func setupSelenium(ctx context.Context) error {
	log.Infof("Starting Selenium Grid")
	cmd := exec.CommandContext(ctx, "docker", "pull", "selenium/standalone-edge:latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error occurred pulling selenium image\n Output: %s\n Error: %s\n", output, err)
		dockerSucceeded = false
	}

	log.Infof("Selenium Image Pull Output : %s\n", output)

	cmd = exec.CommandContext(ctx, "docker", "run", "-d", "-p", "4444:4444", "--name", seleniumContainerName, "--network=host", "--shm-size=2g", "selenium/standalone-edge:latest")
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error occurred starting selenium grid\n Output: %s\n Error: %s\n", output, err)
		dockerSucceeded = false
	}

	log.Infof("Selenium Container Run Output : %s\n", output)

	return err
}

func tearDownSelenium(ctx context.Context) error {
	log.Infof("Stopping Selenium Grid")
	cmd := exec.CommandContext(ctx, "docker", "stop", seleniumContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error occurred stopping selenium container\n Output: %s\n Error: %s\n", output, err)
		dockerSucceeded = false
		return err
	}

	log.Infof("Removing Selenium Grid container")
	cmd = exec.CommandContext(ctx, "docker", "rm", seleniumContainerName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error occurred removing selenium grid container\n Output: %s\n Error: %s\n", output, err)
		dockerSucceeded = false
		return err
	}

	return nil
}

func setup(ctx context.Context) error {
	for _, key := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"CLUSTER",
		"LOCATION",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	var err error
	_env, err = env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

	vnetResourceGroup = os.Getenv("RESOURCEGROUP") // TODO: remove this when we deploy and peer a vnet per cluster create
	if os.Getenv("CI") != "" {
		vnetResourceGroup = os.Getenv("CLUSTER")
	}
	clusterName = os.Getenv("CLUSTER")

	if os.Getenv("CI") != "" { // always create cluster in CI
		cluster, err := cluster.New(log, _env, os.Getenv("CI") != "")
		if err != nil {
			return err
		}

		err = cluster.Create(ctx, vnetResourceGroup, clusterName)
		if err != nil {
			return err
		}
	}

	clusterResourceID = resourceIDFromEnv()

	clients, err = newClientSet(ctx)
	if err != nil {
		return err
	}

	if os.Getenv("AGENT_NAME") != "" {
		// Skip in pipelines for now
		dockerSucceeded = false
	} else {
		cmd := exec.CommandContext(ctx, "which", "docker")
		_, err = cmd.CombinedOutput()
		if err == nil {
			dockerSucceeded = true
		}

		if dockerSucceeded {
			setupSelenium(ctx)
		}
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

	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	if err := setup(context.Background()); err != nil {
		panic(err)
	}
})

var _ = AfterSuite(func() {
	log.Info("AfterSuite")

	// Azure Pipelines will tear down the image if needed
	if dockerSucceeded && os.Getenv("AGENT_NAME") == "" {
		if err := tearDownSelenium(context.Background()); err != nil {
			log.Printf(err.Error())
		}
	}

	if err := done(context.Background()); err != nil {
		panic(err)
	}
})
