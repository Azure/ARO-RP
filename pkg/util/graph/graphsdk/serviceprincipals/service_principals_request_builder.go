package serviceprincipals

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"

	i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

// ServicePrincipalsRequestBuilder provides operations to manage the collection of servicePrincipal entities.
type ServicePrincipalsRequestBuilder struct {
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}

// ServicePrincipalsRequestBuilderGetQueryParameters retrieve a list of servicePrincipal objects.
type ServicePrincipalsRequestBuilderGetQueryParameters struct {
	// Include count of items
	Count *bool `uriparametername:"%24count"`
	// Expand related entities
	// Deprecated: This property is deprecated, use ExpandAsGetExpandQueryParameterType instead
	Expand []string `uriparametername:"%24expand"`
	// Expand related entities
	ExpandAsGetExpandQueryParameterType []GetExpandQueryParameterType `uriparametername:"%24expand"`
	// Filter items by property values
	Filter *string `uriparametername:"%24filter"`
	// Order items by property values
	// Deprecated: This property is deprecated, use OrderbyAsGetOrderbyQueryParameterType instead
	Orderby []string `uriparametername:"%24orderby"`
	// Order items by property values
	OrderbyAsGetOrderbyQueryParameterType []GetOrderbyQueryParameterType `uriparametername:"%24orderby"`
	// Search items by search phrases
	Search *string `uriparametername:"%24search"`
	// Select properties to be returned
	// Deprecated: This property is deprecated, use SelectAsGetSelectQueryParameterType instead
	Select []string `uriparametername:"%24select"`
	// Select properties to be returned
	SelectAsGetSelectQueryParameterType []GetSelectQueryParameterType `uriparametername:"%24select"`
	// Skip the first n items
	Skip *int32 `uriparametername:"%24skip"`
	// Show only the first n items
	Top *int32 `uriparametername:"%24top"`
}

// ServicePrincipalsRequestBuilderGetRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ServicePrincipalsRequestBuilderGetRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
	// Request query parameters
	QueryParameters *ServicePrincipalsRequestBuilderGetQueryParameters
}

// ServicePrincipalsRequestBuilderPostRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ServicePrincipalsRequestBuilderPostRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}

// ByServicePrincipalId provides operations to manage the collection of servicePrincipal entities.
// returns a *ServicePrincipalItemRequestBuilder when successful
func (m *ServicePrincipalsRequestBuilder) ByServicePrincipalId(servicePrincipalId string) *ServicePrincipalItemRequestBuilder {
	urlTplParams := make(map[string]string)
	for idx, item := range m.BaseRequestBuilder.PathParameters {
		urlTplParams[idx] = item
	}
	if servicePrincipalId != "" {
		urlTplParams["servicePrincipal%2Did"] = servicePrincipalId
	}
	return NewServicePrincipalItemRequestBuilderInternal(urlTplParams, m.BaseRequestBuilder.RequestAdapter)
}

// NewServicePrincipalsRequestBuilderInternal instantiates a new ServicePrincipalsRequestBuilder and sets the default values.
func NewServicePrincipalsRequestBuilderInternal(pathParameters map[string]string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ServicePrincipalsRequestBuilder {
	m := &ServicePrincipalsRequestBuilder{
		BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}/servicePrincipals{?%24count,%24expand,%24filter,%24orderby,%24search,%24select,%24skip,%24top}", pathParameters),
	}
	return m
}

// NewServicePrincipalsRequestBuilder instantiates a new ServicePrincipalsRequestBuilder and sets the default values.
func NewServicePrincipalsRequestBuilder(rawUrl string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ServicePrincipalsRequestBuilder {
	urlParams := make(map[string]string)
	urlParams["request-raw-url"] = rawUrl
	return NewServicePrincipalsRequestBuilderInternal(urlParams, requestAdapter)
}

// Get retrieve a list of servicePrincipal objects.
// returns a ServicePrincipalCollectionResponseable when successful
// returns a ODataError error when the service returns a 4XX or 5XX status code
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/serviceprincipal-list?view=graph-rest-1.0
func (m *ServicePrincipalsRequestBuilder) Get(ctx context.Context, requestConfiguration *ServicePrincipalsRequestBuilderGetRequestConfiguration) (i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalCollectionResponseable, error) {
	requestInfo, err := m.ToGetRequestInformation(ctx, requestConfiguration)
	if err != nil {
		return nil, err
	}
	errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings{
		"XXX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
	}
	res, err := m.BaseRequestBuilder.RequestAdapter.Send(ctx, requestInfo, i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.CreateServicePrincipalCollectionResponseFromDiscriminatorValue, errorMapping)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalCollectionResponseable), nil
}

// Post create a new servicePrincipal object.
// returns a ServicePrincipalable when successful
// returns a ODataError error when the service returns a 4XX or 5XX status code
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/serviceprincipal-post-serviceprincipals?view=graph-rest-1.0
func (m *ServicePrincipalsRequestBuilder) Post(ctx context.Context, body i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, requestConfiguration *ServicePrincipalsRequestBuilderPostRequestConfiguration) (i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, error) {
	requestInfo, err := m.ToPostRequestInformation(ctx, body, requestConfiguration)
	if err != nil {
		return nil, err
	}
	errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings{
		"XXX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
	}
	res, err := m.BaseRequestBuilder.RequestAdapter.Send(ctx, requestInfo, i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.CreateServicePrincipalFromDiscriminatorValue, errorMapping)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable), nil
}

// ToGetRequestInformation retrieve a list of servicePrincipal objects.
// returns a *RequestInformation when successful
func (m *ServicePrincipalsRequestBuilder) ToGetRequestInformation(ctx context.Context, requestConfiguration *ServicePrincipalsRequestBuilderGetRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.GET, m.BaseRequestBuilder.UrlTemplate, m.BaseRequestBuilder.PathParameters)
	if requestConfiguration != nil {
		if requestConfiguration.QueryParameters != nil {
			requestInfo.AddQueryParameters(*(requestConfiguration.QueryParameters))
		}
		requestInfo.Headers.AddAll(requestConfiguration.Headers)
		requestInfo.AddRequestOptions(requestConfiguration.Options)
	}
	requestInfo.Headers.TryAdd("Accept", "application/json")
	return requestInfo, nil
}

// ToPostRequestInformation create a new servicePrincipal object.
// returns a *RequestInformation when successful
func (m *ServicePrincipalsRequestBuilder) ToPostRequestInformation(ctx context.Context, body i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, requestConfiguration *ServicePrincipalsRequestBuilderPostRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.POST, "{+baseurl}/servicePrincipals", m.BaseRequestBuilder.PathParameters)
	if requestConfiguration != nil {
		requestInfo.Headers.AddAll(requestConfiguration.Headers)
		requestInfo.AddRequestOptions(requestConfiguration.Options)
	}
	requestInfo.Headers.TryAdd("Accept", "application/json")
	err := requestInfo.SetContentFromParsable(ctx, m.BaseRequestBuilder.RequestAdapter, "application/json", body)
	if err != nil {
		return nil, err
	}
	return requestInfo, nil
}

// WithUrl returns a request builder with the provided arbitrary URL. Using this method means any other path or query parameters are ignored.
// returns a *ServicePrincipalsRequestBuilder when successful
func (m *ServicePrincipalsRequestBuilder) WithUrl(rawUrl string) *ServicePrincipalsRequestBuilder {
	return NewServicePrincipalsRequestBuilder(rawUrl, m.BaseRequestBuilder.RequestAdapter)
}
