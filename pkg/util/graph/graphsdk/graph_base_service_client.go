package graphsdk

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e "github.com/microsoft/kiota-abstractions-go/store"
	i4bcdc892e61ac17e2afc10b5e2b536b29f4fd6c1ad30f4a5a68df47495db3347 "github.com/microsoft/kiota-serialization-form-go"
	i25911dc319edd61cbac496af7eab5ef20b6069a42515e22ec6a9bc97bf598488 "github.com/microsoft/kiota-serialization-json-go"
	i56887720f41ac882814261620b1c8459c4a992a0207af547c4453dd39fabc426 "github.com/microsoft/kiota-serialization-multipart-go"
	i7294a22093d408fdca300f11b81a887d89c47b764af06c8b803e2323973fdb83 "github.com/microsoft/kiota-serialization-text-go"

	ifc7ae6fb75d952477cad00b42c63b11b8c674355828ff1ba0e1b8bd380f51827 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/applications"
	i50842935825402c554412d8c8453e6ff3db97093d4f614fff0d8372d844cb674 "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/serviceprincipals"
)

// GraphBaseServiceClient the main entry point of the SDK, exposes the configuration and the fluent API.
type GraphBaseServiceClient struct {
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}

// Applications provides operations to manage the collection of application entities.
func (m *GraphBaseServiceClient) Applications() *ifc7ae6fb75d952477cad00b42c63b11b8c674355828ff1ba0e1b8bd380f51827.ApplicationsRequestBuilder {
	return ifc7ae6fb75d952477cad00b42c63b11b8c674355828ff1ba0e1b8bd380f51827.NewApplicationsRequestBuilderInternal(m.PathParameters, m.RequestAdapter)
}

// NewGraphBaseServiceClient instantiates a new GraphBaseServiceClient and sets the default values.
func NewGraphBaseServiceClient(requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter, backingStore ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStoreFactory) *GraphBaseServiceClient {
	m := &GraphBaseServiceClient{
		BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}", map[string]string{}),
	}
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultSerializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriterFactory {
		return i25911dc319edd61cbac496af7eab5ef20b6069a42515e22ec6a9bc97bf598488.NewJsonSerializationWriterFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultSerializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriterFactory {
		return i7294a22093d408fdca300f11b81a887d89c47b764af06c8b803e2323973fdb83.NewTextSerializationWriterFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultSerializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriterFactory {
		return i4bcdc892e61ac17e2afc10b5e2b536b29f4fd6c1ad30f4a5a68df47495db3347.NewFormSerializationWriterFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultSerializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriterFactory {
		return i56887720f41ac882814261620b1c8459c4a992a0207af547c4453dd39fabc426.NewMultipartSerializationWriterFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultDeserializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNodeFactory {
		return i25911dc319edd61cbac496af7eab5ef20b6069a42515e22ec6a9bc97bf598488.NewJsonParseNodeFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultDeserializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNodeFactory {
		return i7294a22093d408fdca300f11b81a887d89c47b764af06c8b803e2323973fdb83.NewTextParseNodeFactory()
	})
	i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RegisterDefaultDeserializer(func() i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNodeFactory {
		return i4bcdc892e61ac17e2afc10b5e2b536b29f4fd6c1ad30f4a5a68df47495db3347.NewFormParseNodeFactory()
	})
	if m.RequestAdapter.GetBaseUrl() == "" {
		m.RequestAdapter.SetBaseUrl("https://graph.microsoft.com/v1.0")
	}
	m.PathParameters["baseurl"] = m.RequestAdapter.GetBaseUrl()
	m.RequestAdapter.EnableBackingStore(backingStore)
	return m
}

// ServicePrincipals provides operations to manage the collection of servicePrincipal entities.
func (m *GraphBaseServiceClient) ServicePrincipals() *i50842935825402c554412d8c8453e6ff3db97093d4f614fff0d8372d844cb674.ServicePrincipalsRequestBuilder {
	return i50842935825402c554412d8c8453e6ff3db97093d4f614fff0d8372d844cb674.NewServicePrincipalsRequestBuilderInternal(m.PathParameters, m.RequestAdapter)
}
