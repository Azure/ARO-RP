package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
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

	doc    *api.OpenShiftClusterDocument
	subDoc *api.SubscriptionDocument

	ocDb database.OpenShiftClusters

	interfacesClient          *armnetwork.InterfacesClient
	loadBalancerClient        *armnetwork.LoadBalancersClient
	resourceSKUsClient        *armcompute.ResourceSKUsClient
	privateLinkServicesClient *armnetwork.PrivateLinkServicesClient
	registriesClient          *armcontainerregistry.RegistriesClient
	tokensClient              *armcontainerregistry.TokensClient

	resultMessage string
}

var _ mimo.TaskContext = &fakeTestContext{}

type Option func(*fakeTestContext)

func WithClientHelper(ch clienthelper.Interface) Option {
	return func(ftc *fakeTestContext) {
		ftc.ch = ch
	}
}

func WithOpenShiftClusterDocument(oc *api.OpenShiftClusterDocument) Option {
	return func(ftc *fakeTestContext) {
		ftc.doc = oc
	}
}

func WithSubscriptionDocument(doc *api.SubscriptionDocument) Option {
	return func(ftc *fakeTestContext) {
		ftc.subDoc = doc
	}
}

func WithLoadBalancersClient(c armnetwork.LoadBalancersClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.loadBalancerClient = &c
	}
}

func WithResourceSKUsClient(c armcompute.ResourceSKUsClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.resourceSKUsClient = &c
	}
}

func WithPrivateLinkServicesClient(c armnetwork.PrivateLinkServicesClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.privateLinkServicesClient = &c
	}
}

func WithInterfacesClient(c armnetwork.InterfacesClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.interfacesClient = &c
	}
}

func WithTokensClient(c armcontainerregistry.TokensClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.tokensClient = &c
	}
}

func WithRegistriesClient(c armcontainerregistry.RegistriesClient) Option {
	return func(ftc *fakeTestContext) {
		ftc.registriesClient = &c
	}
}

func WithOpenShiftDatabase(d database.OpenShiftClusters) Option {
	return func(ftc *fakeTestContext) {
		ftc.ocDb = d
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

func (t *fakeTestContext) GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument {
	if t.doc == nil {
		panic("didn't set up OpenShiftClusterDocument in test")
	}
	return t.doc
}

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

// OpenShiftCluster
func (t *fakeTestContext) GetClusterUUID() string {
	if t.doc == nil {
		panic("didn't set up OpenShiftClusterDocument in test")
	}
	return t.doc.ID
}

func (t *fakeTestContext) GetOpenShiftClusterProperties() api.OpenShiftClusterProperties {
	if t.doc == nil {
		panic("didn't set up OpenShiftClusterDocument in test")
	}
	return t.doc.OpenShiftCluster.Properties
}

// Result
func (t *fakeTestContext) SetResultMessage(s string) {
	t.resultMessage = s
}

// GetResultMessage is used for verification in tests, it does not appear on the
// TestContext interface and cannot be called by Tasks.
func (t *fakeTestContext) GetResultMessage() string {
	return t.resultMessage
}

// Azure Clients
func (t *fakeTestContext) LoadBalancersClient() (armnetwork.LoadBalancersClient, error) {
	if t.loadBalancerClient == nil {
		return nil, fmt.Errorf("no LB client provided")
	}
	return *t.loadBalancerClient, nil
}

func (t *fakeTestContext) ResourceSKUsClient() (armcompute.ResourceSKUsClient, error) {
	if t.resourceSKUsClient == nil {
		return nil, fmt.Errorf("no ResourceSKUs client provided")
	}
	return *t.resourceSKUsClient, nil
}

func (t *fakeTestContext) PrivateLinkServicesClient() (armnetwork.PrivateLinkServicesClient, error) {
	if t.privateLinkServicesClient == nil {
		return nil, fmt.Errorf("no PLS client provided")
	}
	return *t.privateLinkServicesClient, nil
}

func (t *fakeTestContext) InterfacesClient() (armnetwork.InterfacesClient, error) {
	if t.interfacesClient == nil {
		return nil, fmt.Errorf("no armnetwork.InterfacesClient provided")
	}
	return *t.interfacesClient, nil
}

func (t *fakeTestContext) RegistriesClient() (armcontainerregistry.RegistriesClient, error) {
	if t.registriesClient == nil {
		return nil, fmt.Errorf("no armcontainerregistry.RegistriesClient provided")
	}
	return *t.registriesClient, nil
}

func (t *fakeTestContext) TokensClient() (armcontainerregistry.TokensClient, error) {
	if t.tokensClient == nil {
		return nil, fmt.Errorf("no armcontainerregistry.TokensClient provided")
	}
	return *t.tokensClient, nil
}
