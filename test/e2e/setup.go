package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tebeka/selenium"

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
	redhatopenshift20210901preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2021-09-01-preview/redhatopenshift"
	redhatopenshift20220401 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-04-01/redhatopenshift"
	redhatopenshift20220904 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2022-09-04/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

type clientSet struct {
	OpenshiftClustersv20200430        redhatopenshift20200430.OpenShiftClustersClient
	Operationsv20200430               redhatopenshift20200430.OperationsClient
	OpenshiftClustersv20210901preview redhatopenshift20210901preview.OpenShiftClustersClient
	Operationsv20210901preview        redhatopenshift20210901preview.OperationsClient
	OpenshiftClustersv20220401        redhatopenshift20220401.OpenShiftClustersClient
	Operationsv20220401               redhatopenshift20220401.OperationsClient
	OpenshiftClustersv20220904        redhatopenshift20220904.OpenShiftClustersClient
	Operationsv20220904               redhatopenshift20220904.OperationsClient

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

func generateSession(ctx context.Context, log *logrus.Entry) (string, error) {

	const (
		SessionName        = "session"
		SessionKeyExpires  = "expires"
		sessionKeyState    = "state"
		SessionKeyUsername = "user_name"
		SessionKeyGroups   = "groups"
		username           = "testuser"
		groups             = ""
	)

	flag.Parse()

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return "", err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return "", err
	}

	portalKeyvaultURI, err := keyvault.URI(_env, env.PortalKeyvaultSuffix)
	if err != nil {
		return "", err
	}

	portalKeyvault := keyvault.NewManager(msiKVAuthorizer, portalKeyvaultURI)

	sessionKey, err := portalKeyvault.GetBase64Secret(ctx, env.PortalServerSessionKeySecretName, "")
	if err != nil {
		return "", err
	}

	store := sessions.NewCookieStore(sessionKey)

	store.MaxAge(0)
	store.Options.Secure = true
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteLaxMode

	session := sessions.NewSession(store, SessionName)
	opts := *store.Options
	session.Options = &opts

	session.Values[SessionKeyUsername] = username
	session.Values[SessionKeyGroups] = strings.Split(groups, ",")
	session.Values[SessionKeyExpires] = time.Now().Add(time.Hour)

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		store.Codecs...)
	if err != nil {
		log.Infof(err.Error())
		return "", err
	}

	// encoded
	log.Infof("session=%s", encoded)

	return encoded, nil
}

func adminPortalSessionSetup() *selenium.WebDriver {
	const (
		port = 4444
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	caps := selenium.Capabilities{
		"browserName":         "MicrosoftEdge",
		"acceptInsecureCerts": true,
	}
	wd := selenium.WebDriver(nil)

	var err error

	for {
		wd, err = selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
		if wd != nil || ctx.Err() != nil {
			break
		}
	}

	if err != nil {
		panic(err)
	}

	// Navigate to the simple playground interface.
	if err := wd.Get(os.Getenv("PORTAL_HOSTNAME") + ":8444/api/info"); err != nil {
		panic(err)
	}

	log := utillog.GetLogger()

	gob.Register(time.Time{})

	session, err := generateSession(context.Background(), log)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Session: %s", session)

	cookie := &selenium.Cookie{
		Name:   "session",
		Value:  session,
		Expiry: math.MaxUint32,
	}

	if err := wd.AddCookie(cookie); err != nil {
		panic(err)
	}

	tests, err := wd.GetCookies()
	if err != nil {
		panic(err)
	}

	for _, test := range tests {
		fmt.Printf("Name : %s\n Value : %s\n Domain : %s\n Path : %s\n Secure: %s\n Expiry : %d\n",
			test.Name,
			test.Value,
			test.Domain,
			test.Path,
			strconv.FormatBool(test.Secure),
			test.Expiry)
	}

	if err := wd.Get(os.Getenv("PORTAL_HOSTNAME") + ":8444/v2"); err != nil {
		panic(err)
	}

	return &wd
}

func adminPortalSessionTearDown() {
	log.Infof("Stopping Selenium Grid")
	cmd := exec.Command("docker", "rm", "--force", "selenium-edge-standalone")

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatalf("Error occurred stopping selenium grid\n Output: %s\n Error: %s\n", output, err)
	}
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
		OpenshiftClustersv20200430:        redhatopenshift20200430.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20200430:               redhatopenshift20200430.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClustersv20210901preview: redhatopenshift20210901preview.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20210901preview:        redhatopenshift20210901preview.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClustersv20220401:        redhatopenshift20220401.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20220401:               redhatopenshift20220401.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),

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

	log.Infof("Starting Selenium Grid")
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "-p", "4444:4444", "--name", "selenium-edge-standalone", "--network=host", "--shm-size=2g", "selenium/standalone-edge:latest")

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatalf("Error occurred starting selenium grid\n Output: %s\n Error: %s\n", output, err)
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

	adminPortalSessionTearDown()

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

	if err := done(context.Background()); err != nil {
		panic(err)
	}
})
