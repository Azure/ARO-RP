package applications

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"

	i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

// ItemAddPasswordRequestBuilder provides operations to call the addPassword method.
type ItemAddPasswordRequestBuilder struct {
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}

// ItemAddPasswordRequestBuilderPostRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ItemAddPasswordRequestBuilderPostRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}

// NewItemAddPasswordRequestBuilderInternal instantiates a new AddPasswordRequestBuilder and sets the default values.
func NewItemAddPasswordRequestBuilderInternal(pathParameters map[string]string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ItemAddPasswordRequestBuilder {
	m := &ItemAddPasswordRequestBuilder{
		BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}/applications/{application%2Did}/addPassword", pathParameters),
	}
	return m
}

// NewItemAddPasswordRequestBuilder instantiates a new AddPasswordRequestBuilder and sets the default values.
func NewItemAddPasswordRequestBuilder(rawUrl string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ItemAddPasswordRequestBuilder {
	urlParams := make(map[string]string)
	urlParams["request-raw-url"] = rawUrl
	return NewItemAddPasswordRequestBuilderInternal(urlParams, requestAdapter)
}

// Post adds a strong password or secret to an application.
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/application-addpassword?view=graph-rest-1.0
func (m *ItemAddPasswordRequestBuilder) Post(ctx context.Context, body ItemAddPasswordPostRequestBodyable, requestConfiguration *ItemAddPasswordRequestBuilderPostRequestConfiguration) (i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable, error) {
	requestInfo, err := m.ToPostRequestInformation(ctx, body, requestConfiguration)
	if err != nil {
		return nil, err
	}
	errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings{
		"4XX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
		"5XX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
	}
	res, err := m.RequestAdapter.Send(ctx, requestInfo, i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.CreatePasswordCredentialFromDiscriminatorValue, errorMapping)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable), nil
}

// ToPostRequestInformation adds a strong password or secret to an application.
func (m *ItemAddPasswordRequestBuilder) ToPostRequestInformation(ctx context.Context, body ItemAddPasswordPostRequestBodyable, requestConfiguration *ItemAddPasswordRequestBuilderPostRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformation()
	requestInfo.UrlTemplate = m.UrlTemplate
	requestInfo.PathParameters = m.PathParameters
	requestInfo.Method = i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.POST
	requestInfo.Headers.Add("Accept", "application/json")
	err := requestInfo.SetContentFromParsable(ctx, m.RequestAdapter, "application/json", body)
	if err != nil {
		return nil, err
	}
	if requestConfiguration != nil {
		requestInfo.Headers.AddAll(requestConfiguration.Headers)
		requestInfo.AddRequestOptions(requestConfiguration.Options)
	}
	return requestInfo, nil
}
