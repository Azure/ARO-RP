package applications

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"

	i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

// ItemRemovePasswordRequestBuilder provides operations to call the removePassword method.
type ItemRemovePasswordRequestBuilder struct {
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}

// ItemRemovePasswordRequestBuilderPostRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type ItemRemovePasswordRequestBuilderPostRequestConfiguration struct {
	// Request headers
	Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
	// Request options
	Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}

// NewItemRemovePasswordRequestBuilderInternal instantiates a new RemovePasswordRequestBuilder and sets the default values.
func NewItemRemovePasswordRequestBuilderInternal(pathParameters map[string]string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ItemRemovePasswordRequestBuilder {
	m := &ItemRemovePasswordRequestBuilder{
		BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}/applications/{application%2Did}/removePassword", pathParameters),
	}
	return m
}

// NewItemRemovePasswordRequestBuilder instantiates a new RemovePasswordRequestBuilder and sets the default values.
func NewItemRemovePasswordRequestBuilder(rawUrl string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter) *ItemRemovePasswordRequestBuilder {
	urlParams := make(map[string]string)
	urlParams["request-raw-url"] = rawUrl
	return NewItemRemovePasswordRequestBuilderInternal(urlParams, requestAdapter)
}

// Post remove a password from an application.
// [Find more info here]
//
// [Find more info here]: https://learn.microsoft.com/graph/api/application-removepassword?view=graph-rest-1.0
func (m *ItemRemovePasswordRequestBuilder) Post(ctx context.Context, body ItemRemovePasswordPostRequestBodyable, requestConfiguration *ItemRemovePasswordRequestBuilderPostRequestConfiguration) error {
	requestInfo, err := m.ToPostRequestInformation(ctx, body, requestConfiguration)
	if err != nil {
		return err
	}
	errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings{
		"4XX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
		"5XX": i590dfc7f28a1fc5720c211d996119093307169ae10220ddded8912d222cbd376.CreateODataErrorFromDiscriminatorValue,
	}
	err = m.BaseRequestBuilder.RequestAdapter.SendNoContent(ctx, requestInfo, errorMapping)
	if err != nil {
		return err
	}
	return nil
}

// ToPostRequestInformation remove a password from an application.
func (m *ItemRemovePasswordRequestBuilder) ToPostRequestInformation(ctx context.Context, body ItemRemovePasswordPostRequestBodyable, requestConfiguration *ItemRemovePasswordRequestBuilderPostRequestConfiguration) (*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
	requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformation()
	requestInfo.UrlTemplate = m.BaseRequestBuilder.UrlTemplate
	requestInfo.PathParameters = m.BaseRequestBuilder.PathParameters
	requestInfo.Method = i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.POST
	err := requestInfo.SetContentFromParsable(ctx, m.BaseRequestBuilder.RequestAdapter, "application/json", body)
	if err != nil {
		return nil, err
	}
	if requestConfiguration != nil {
		requestInfo.Headers.AddAll(requestConfiguration.Headers)
		requestInfo.AddRequestOptions(requestConfiguration.Options)
	}
	return requestInfo, nil
}
