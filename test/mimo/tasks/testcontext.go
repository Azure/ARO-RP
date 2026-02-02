package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

type fakeTestContext struct {
	context.Context
	now func() time.Time
	env env.Interface
	ch  clienthelper.Interface
	log *logrus.Entry

	ocDatabase database.OpenShiftClusters

	clusterUUID       string
	clusterResourceID string
	tenantID          string
	properties        api.OpenShiftClusterProperties
	doc               *api.OpenShiftClusterDocument

	interfacesClient          armnetwork.InterfacesClient
	loadBalancerClient        armnetwork.LoadBalancersClient
	resourceSKUsClient        armcompute.ResourceSKUsClient
	privateLinkServicesClient armnetwork.PrivateLinkServicesClient

	resultMessage string
}

var _ mimo.TaskContextWithAzureClients = &fakeTestContext{}

type Option func(*fakeTestContext)

func WithClientHelper(ch clienthelper.Interface) Option {
	return func(ftc *fakeTestContext) {
		ftc.ch = ch
	}
}

func WithOpenShiftClusterDocument(oc *api.OpenShiftClusterDocument) Option {
	return func(ftc *fakeTestContext) {
		ftc.clusterUUID = oc.ID
		ftc.clusterResourceID = oc.OpenShiftCluster.ID
		ftc.properties = oc.OpenShiftCluster.Properties
		ftc.doc = oc
	}
}

func WithOpenShiftClusterProperties(uuid string, oc api.OpenShiftClusterProperties) Option {
	return func(ftc *fakeTestContext) {
		ftc.clusterUUID = uuid
		ftc.properties = oc
	}
}

func WithOpenShiftDatabase(d database.OpenShiftClusters) Option {
	return func(ftc *fakeTestContext) {
		ftc.ocDatabase = d
	}
}

func WithLoadBalancersClient(c armnetwork.LoadBalancersClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.loadBalancerClient = c
	}
}

func WithResourceSKUsClient(c armcompute.ResourceSKUsClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.resourceSKUsClient = c
	}
}

func WithPrivateLinkServicesClient(c armnetwork.PrivateLinkServicesClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.privateLinkServicesClient = c
	}
}

func WithInterfacesClient(c armnetwork.InterfacesClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.interfacesClient = c
	}
}

func NewFakeTestContext(ctx context.Context, env env.Interface, log *logrus.Entry, now func() time.Time, o ...Option) *fakeTestContext {
	ftc := &fakeTestContext{
		Context: ctx,
		env:     env,
		log:     log,
		now:     now,
	}
	for _, i := range o {
		i(ftc)
	}
	return ftc
}

func (t *fakeTestContext) LocalFpAuthorizer() (autorest.Authorizer, error) {
	myAuthorizer := autorest.NullAuthorizer{}
	return myAuthorizer, nil
}
func (t *fakeTestContext) GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument {
	if t.doc == nil {
		panic("didn't set up OpenShiftClusterDocument in test")
	}
	return t.doc
}

// handle

func (t *fakeTestContext) Environment() env.Interface {
	return t.env
}

func (t *fakeTestContext) ClientHelper() (clienthelper.Interface, error) {
	if t.ch == nil {
		return nil, fmt.Errorf("missing clienthelper")
	}
	return t.ch, nil
}

func (t *fakeTestContext) Log() *logrus.Entry {
	return t.log
}

func (t *fakeTestContext) Now() time.Time {
	return t.now()
}

// Subscription
func (t *fakeTestContext) GetTenantID() string {
	if t.tenantID == "" {
		panic("didn't set up tenantID in test")
	}
	return t.tenantID
}

// OpenShiftCluster
func (t *fakeTestContext) GetClusterUUID() string {
	if t.clusterUUID == "" {
		panic("didn't set up openshiftcluster in test")
	}
	return t.clusterUUID
}

func (t *fakeTestContext) GetOpenShiftClusterProperties() api.OpenShiftClusterProperties {
	if t.clusterUUID == "" {
		panic("didn't set up openshiftcluster in test")
	}
	return t.properties
}

func (t *fakeTestContext) PatchOpenShiftClusterDocument(ctx context.Context, f database.OpenShiftClusterDocumentMutator) (*api.OpenShiftClusterDocument, error) {
	return t.ocDatabase.PatchWithLease(ctx, t.doc.Key, f)
}

// Result
func (t *fakeTestContext) SetResultMessage(s string) {
	t.resultMessage = s
}

func (t *fakeTestContext) GetResultMessage() string {
	return t.resultMessage
}

// WithAzureClients
func (t *fakeTestContext) LoadBalancersClient() (armnetwork.LoadBalancersClient, error) {
	if t.loadBalancerClient == nil {
		return nil, fmt.Errorf("no LB client provided")
	}

	return t.loadBalancerClient, nil
}

func (t *fakeTestContext) ResourceSKUsClient() (armcompute.ResourceSKUsClient, error) {
	if t.resourceSKUsClient == nil {
		return nil, fmt.Errorf("no ResourceSKUs client provided")
	}

	return t.resourceSKUsClient, nil
}

func (t *fakeTestContext) PrivateLinkServicesClient() (armnetwork.PrivateLinkServicesClient, error) {
	if t.privateLinkServicesClient == nil {
		return nil, fmt.Errorf("no PLS client provided")
	}

	return t.privateLinkServicesClient, nil
}

func (t *fakeTestContext) InterfacesClient() (armnetwork.InterfacesClient, error) {
	if t.interfacesClient == nil {
		return nil, fmt.Errorf("no armnetwork.InterfacesClient provided")
	}

	return t.interfacesClient, nil
}
