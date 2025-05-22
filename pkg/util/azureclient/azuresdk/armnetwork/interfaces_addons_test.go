package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	fakearmnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6/fake"
	"github.com/Azure/go-autorest/autorest/to"
)

var (
	subscription      = "0000000-0000-0000-0000-000000000000"
	ctx               = context.Background()
	resourceGroupName = "rg"
	listOptions       = &armnetwork.InterfacesClientListOptions{}
	noPagerResults    = make([]*armnetwork.Interface, 0)
	pagerResults      = []*armnetwork.Interface{
		{
			Name:       to.StringPtr("interface1"),
			Properties: &armnetwork.InterfacePropertiesFormat{},
		},
		{
			Name:       to.StringPtr("interface2"),
			Properties: &armnetwork.InterfacePropertiesFormat{},
		},
	}
)

func Test_interfacesClient_List(t *testing.T) {
	tests := []struct {
		name          string
		clientOptions arm.ClientOptions
		wantResult    []*armnetwork.Interface
		wantErr       string
	}{
		{
			name:          "API auth error",
			clientOptions: *clientOptions(ifServerAuthError()),
			wantErr:       "fake API auth error",
		},
		{
			name:          "No results from pager",
			clientOptions: *clientOptions(ifServer(noPagerResults)),
		},
		{
			name:          "Results from pager",
			clientOptions: *clientOptions(ifServer(pagerResults)),
			wantResult:    pagerResults,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewInterfacesClient(subscription, &fakeazcore.TokenCredential{}, &tt.clientOptions)
			if err != nil {
				t.Errorf("NewInterfacesClient() error = %v", err)
			}

			gotResult, err := c.List(ctx, resourceGroupName, listOptions)
			if err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("List() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func ifServerAuthError() fakearmnetwork.InterfacesServer {
	return fakearmnetwork.InterfacesServer{
		NewListPager: func(resourceGroupName string, options *armnetwork.InterfacesClientListOptions) (resp fakeazcore.PagerResponder[armnetwork.InterfacesClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armnetwork.InterfacesClientListResponse]{}
			pagerResponse.AddResponseError(http.StatusForbidden, "fake API auth error")

			return pagerResponse
		},
	}
}

func ifServer(results []*armnetwork.Interface) fakearmnetwork.InterfacesServer {
	return fakearmnetwork.InterfacesServer{
		NewListPager: func(resourceGroupName string, options *armnetwork.InterfacesClientListOptions) (resp fakeazcore.PagerResponder[armnetwork.InterfacesClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armnetwork.InterfacesClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armnetwork.InterfacesClientListResponse{
				InterfaceListResult: armnetwork.InterfaceListResult{
					Value: results,
				},
			}, nil)

			return pagerResponse
		},
	}
}

func clientOptions(server fakearmnetwork.InterfacesServer) *arm.ClientOptions {
	return &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakearmnetwork.NewInterfacesServerTransport(&server),
		},
	}
}
