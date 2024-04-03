package serviceprincipals

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"

	i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	iac4789eacca6f59378676f024ff577cfe06db0286ce9db9310b1aba1c2fb20c3 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/serviceprincipals/item"
)

// ServicePrincipalItemRequestBuilder provides operations to manage the collection of servicePrincipal entities.
type ServicePrincipalItemRequestBuilder struct {
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}

// ServicePrincipalItemRequestBuilderDeleteRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ServicePrincipalItemRequestBuilderDeleteRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}

// ServicePrincipalItemRequestBuilderGetQueryParameters retrieve the properties and relationships of a servicePrincipal object.
type ServicePrincipalItemRequestBuilderGetQueryParameters struct {
	// Expand related entities
	// Deprecated: This property is deprecated, use ExpandAsGetExpandQueryParameterType instead
	Expand []string `uriparametername:"%24expand"`
	// Expand related entities
	ExpandAsGetExpandQueryParameterType []iac4789eacca6f59378676f024ff577cfe06db0286ce9db9310b1aba1c2fb20c3.GetExpandQueryParameterType `uriparametername:"%24expand"`
	// Select properties to be returned
	// Deprecated: This property is deprecated, use SelectAsGetSelectQueryParameterType instead
	Select []string `uriparametername:"%24select"`
	// Select properties to be returned
	SelectAsGetSelectQueryParameterType []iac4789eacca6f59378676f024ff577cfe06db0286ce9db9310b1aba1c2fb20c3.GetSelectQueryParameterType `uriparametername:"%24select"`
}

// ServicePrincipalItemRequestBuilderGetRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ServicePrincipalItemRequestBuilderGetRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
	// Request query parameters
	QueryParameters *ServicePrincipalItemRequestBuilderGetQueryParameters
}

// ServicePrincipalItemRequestBuilderPatchRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ServicePrincipalItemRequestBuilderPatchRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}

// NewServicePrincipalItemRequestBuilderInternal instantiates a new ServicePrincipalItemRequestBuilder and sets the default values.
func NewServicePrincipalItemRequestBuilderInternal(pathParameters map[string]string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ServicePrincipalItemRequestBuilder {
	m := &ServicePrincipalItemRequestBuilder{
		BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}/servicePrincipals/{servicePrincipal%2Did}{?%24expand,%24select}", pathParameters),
	}
	return m
}

// NewServicePrincipalItemRequestBuilder instantiates a new ServicePrincipalItemRequestBuilder and sets the default values.
func NewServicePrincipalItemRequestBuilder(rawUrl string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ServicePrincipalItemRequestBuilder {
	urlParams := make(map[string]string)
	urlParams["request-raw-url"] = rawUrl
	return NewServicePrincipalItemRequestBuilderInternal(urlParams, requestAdapter)
}

// Delete delete a servicePrincipal object.
// returns a ODataError error when the service returns a 4XX or 5XX status code
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/serviceprincipal-delete?view=graph-rest-1.0
func (m *ServicePrincipalItemRequestBuilder) Delete(ctx context.Context, requestConfiguration *ServicePrincipalItemRequestBuilderDeleteRequestConfiguration) error {
	requestInfo, err := m.ToDeleteRequestInformation(ctx, requestConfiguration)
	if err != nil {
		return err
	}
	errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings{
		"XXX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
	}
	err = m.BaseRequestBuilder.RequestAdapter.SendNoContent(ctx, requestInfo, errorMapping)
	if err != nil {
		return err
	}
	return nil
}

// Get retrieve the properties and relationships of a servicePrincipal object.
// returns a ServicePrincipalable when successful
// returns a ODataError error when the service returns a 4XX or 5XX status code
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/serviceprincipal-get?view=graph-rest-1.0
func (m *ServicePrincipalItemRequestBuilder) Get(ctx context.Context, requestConfiguration *ServicePrincipalItemRequestBuilderGetRequestConfiguration) (i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, error) {
	requestInfo, err := m.ToGetRequestInformation(ctx, requestConfiguration)
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

// Patch update entity in servicePrincipals
// returns a ServicePrincipalable when successful
// returns a ODataError error when the service returns a 4XX or 5XX status code
func (m *ServicePrincipalItemRequestBuilder) Patch(ctx context.Context, body i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, requestConfiguration *ServicePrincipalItemRequestBuilderPatchRequestConfiguration) (i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, error) {
	requestInfo, err := m.ToPatchRequestInformation(ctx, body, requestConfiguration)
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

// ToDeleteRequestInformation delete a servicePrincipal object.
// returns a *RequestInformation when successful
func (m *ServicePrincipalItemRequestBuilder) ToDeleteRequestInformation(ctx context.Context, requestConfiguration *ServicePrincipalItemRequestBuilderDeleteRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.DELETE, "{+baseurl}/servicePrincipals/{servicePrincipal%2Did}", m.BaseRequestBuilder.PathParameters)
	if requestConfiguration != nil {
		requestInfo.Headers.AddAll(requestConfiguration.Headers)
		requestInfo.AddRequestOptions(requestConfiguration.Options)
	}
	requestInfo.Headers.TryAdd("Accept", "application/json")
	return requestInfo, nil
}

// ToGetRequestInformation retrieve the properties and relationships of a servicePrincipal object.
// returns a *RequestInformation when successful
func (m *ServicePrincipalItemRequestBuilder) ToGetRequestInformation(ctx context.Context, requestConfiguration *ServicePrincipalItemRequestBuilderGetRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
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

// ToPatchRequestInformation update entity in servicePrincipals
// returns a *RequestInformation when successful
func (m *ServicePrincipalItemRequestBuilder) ToPatchRequestInformation(ctx context.Context, body i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.ServicePrincipalable, requestConfiguration *ServicePrincipalItemRequestBuilderPatchRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.PATCH, "{+baseurl}/servicePrincipals/{servicePrincipal%2Did}", m.BaseRequestBuilder.PathParameters)
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
// returns a *ServicePrincipalItemRequestBuilder when successful
func (m *ServicePrincipalItemRequestBuilder) WithUrl(rawUrl string) *ServicePrincipalItemRequestBuilder {
	return NewServicePrincipalItemRequestBuilder(rawUrl, m.BaseRequestBuilder.RequestAdapter)
}
