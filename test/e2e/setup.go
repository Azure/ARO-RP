package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"  //nolint
	. "github.com/onsi/gomega"     //nolint
	. "github.com/tebeka/selenium" //nolint

	"github.com/Azure/go-autorest/autorest/azure/auth"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/Azure/ARO-RP/pkg/env"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	redhatopenshift20200430 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2020-04-30/redhatopenshift"
	redhatopenshift20220401 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-04-01/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

const seleniumContainerName = "selenium-edge-standalone"

type clientSet struct {
	OpenshiftClustersv20200430 redhatopenshift20200430.OpenShiftClustersClient
	Operationsv20200430        redhatopenshift20200430.OperationsClient
	OpenshiftClustersv20220401 redhatopenshift20220401.OpenShiftClustersClient

	VirtualMachines       compute.VirtualMachinesClient
	Resources             features.ResourcesClient
	VirtualNetworks       network.VirtualNetworksClient
	DiskEncryptionSets    compute.DiskEncryptionSetsClient
	Disks                 compute.DisksClient
	NetworkSecurityGroups network.SecurityGroupsClient
	Subnet                network.SubnetsClient

	RestConfig    *rest.Config
	Kubernetes    kubernetes.Interface
	MachineAPI    machineclient.Interface
	MachineConfig mcoclient.Interface
	AROClusters   aroclient.Interface
	ConfigClient  configclient.Interface
	Project       projectclient.Interface
}

var (
	log               *logrus.Entry
	_env              env.Core
	vnetResourceGroup string
	clusterName       string
	clients           *clientSet
)

func skipIfNotInDevelopmentEnv() {
	if !_env.IsLocalDevelopmentMode() {
		Skip("skipping tests in non-development environment")
	}
}

func SaveScreenshotAndExit(wd WebDriver, e error) {
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

func adminPortalSessionSetup() (string, *WebDriver) {
	const (
		hubPort  = 4444
		hostPort = 8444
	)

	os.Setenv("SE_SESSION_REQUEST_TIMEOUT", "9000")

	caps := Capabilities{
		"browserName":         "MicrosoftEdge",
		"acceptInsecureCerts": true,
	}
	wd := WebDriver(nil)

	_, err := url.ParseRequestURI(fmt.Sprintf("https://localhost:%d", hubPort))
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		wd, err = NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", hubPort))
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

	gob.Register(time.Time{})

	// Navigate to the simple playground interface.
	host, exists := os.LookupEnv("PORTAL_HOSTNAME")
	if !exists {
		host = fmt.Sprintf("https://localhost:%d", hostPort)
	}

	if err := wd.Get(host + "/api/info"); err != nil {
		log.Infof("Could not get to %s. With error : %s", host, err.Error())
	}

	cmd := exec.Command("go", "run", "./hack/portalauth", "-username", "test", "-groups", "$AZURE_PORTAL_ELEVATED_GROUP_IDS", "2>", "/dev/null")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error occurred creating session cookie\n Output: %s\n Error: %s\n", output, err)
	}

	os.Setenv("SESSION", string(output))

	log.Infof("Session Output : %s\n", os.Getenv("SESSION"))

	cookie := &Cookie{
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

	return &clientSet{
		OpenshiftClustersv20200430: redhatopenshift20200430.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20200430:        redhatopenshift20200430.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClustersv20220401: redhatopenshift20220401.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		VirtualMachines:       compute.NewVirtualMachinesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Resources:             features.NewResourcesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		VirtualNetworks:       network.NewVirtualNetworksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Disks:                 compute.NewDisksClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		DiskEncryptionSets:    compute.NewDiskEncryptionSetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Subnet:                network.NewSubnetsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		NetworkSecurityGroups: network.NewSecurityGroupsClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		RestConfig:    restconfig,
		Kubernetes:    cli,
		MachineAPI:    machineapicli,
		MachineConfig: mcocli,
		AROClusters:   arocli,
		Project:       projectcli,
		ConfigClient:  configcli,
	}, nil
}

func setupSelenium(ctx context.Context) error {
	log.Infof("Starting Selenium Grid")
	cmd := exec.CommandContext(ctx, "docker", "pull", "selenium/standalone-edge:latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error occurred pulling selenium image\n Output: %s\n Error: %s\n", output, err)
	}

	log.Infof("Selenium Image Pull Output : %s\n", output)

	cmd = exec.CommandContext(ctx, "docker", "run", "-d", "-p", "4444:4444", "--name", seleniumContainerName, "--network=host", "--shm-size=2g", "selenium/standalone-edge:latest")
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error occurred starting selenium grid\n Output: %s\n Error: %s\n", output, err)
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
	}

	log.Infof("Removing Selenium Grid container")
	cmd = exec.CommandContext(ctx, "docker", "rm", seleniumContainerName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error occurred removing selenium grid container\n Output: %s\n Error: %s\n", output, err)
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

	clients, err = newClientSet(ctx)
	if err != nil {
		return err
	}

	setupSelenium(ctx)

	return nil
}

func tearDown(ctx context.Context) error {
	return tearDownSelenium(context.Background())
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

	if err := tearDown(context.Background()); err != nil {
		panic(err)
	}
})
