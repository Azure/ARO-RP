package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/test/clients/openshift"
)

type ClientSet struct {
	log *logrus.Entry

	openshiftclusters redhatopenshift.OpenShiftClustersClient
	openshiftclient   *openshift.Client
}

var (
	Clients *ClientSet
)

func newClientSet(log *logrus.Entry) (*ClientSet, error) {
	conf := auth.NewClientCredentialsConfig(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"))
	authorizer, err := conf.Authorizer()
	if err != nil {
		return nil, err
	}

	cs := &ClientSet{
		log:               log,
		openshiftclusters: redhatopenshift.NewOpenShiftClustersClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
	}
	d, err := ioutil.ReadFile("../../admin.kubeconfig")
	if err != nil {
		return nil, err
	}
	var adminKubeconfig *v1.Config
	json.Unmarshal(d, &adminKubeconfig)
	if err != nil {
		return nil, err
	}
	cs.openshiftclient, err = openshift.NewAdminClient(log, adminKubeconfig)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

var _ = BeforeSuite(func() {
	logrus.SetOutput(GinkgoWriter)
	logger := utillog.GetLogger()
	for _, key := range []string{
		"AZURE_SUBSCRIPTION_ID", "AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET",
		"CLUSTER", "RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			panic(fmt.Sprintf("environment variable %q unset", key))
		}
	}
	var err error
	Clients, err = newClientSet(logger)
	if err != nil {
		panic(err)
	}
})
