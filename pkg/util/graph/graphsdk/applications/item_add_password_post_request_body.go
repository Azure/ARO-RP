package applications

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e "github.com/microsoft/kiota-abstractions-go/store"

	i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
)

// ItemAddPasswordPostRequestBody
type ItemAddPasswordPostRequestBody struct {
	// Stores model information.
	backingStore ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
}

// NewItemAddPasswordPostRequestBody instantiates a new ItemAddPasswordPostRequestBody and sets the default values.
func NewItemAddPasswordPostRequestBody() *ItemAddPasswordPostRequestBody {
	m := &ItemAddPasswordPostRequestBody{}
	m.backingStore = ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStoreFactoryInstance()
	return m
}

// CreateItemAddPasswordPostRequestBodyFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateItemAddPasswordPostRequestBodyFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	return NewItemAddPasswordPostRequestBody(), nil
}

// GetBackingStore gets the backingStore property value. Stores model information.
func (m *ItemAddPasswordPostRequestBody) GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore {
	return m.backingStore
}

// GetFieldDeserializers the deserialization information for the current model
func (m *ItemAddPasswordPostRequestBody) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := make(map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error)
	res["passwordCredential"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetObjectValue(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.CreatePasswordCredentialFromDiscriminatorValue)
		if err != nil {
			return err
		}
		if val != nil {
			m.SetPasswordCredential(val.(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable))
		}
		return nil
	}
	return res
}

// GetPasswordCredential gets the passwordCredential property value. The passwordCredential property
func (m *ItemAddPasswordPostRequestBody) GetPasswordCredential() i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable {
	val, err := m.GetBackingStore().Get("passwordCredential")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable)
	}
	return nil
}

// Serialize serializes information the current object
func (m *ItemAddPasswordPostRequestBody) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	{
		err := writer.WriteObjectValue("passwordCredential", m.GetPasswordCredential())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBackingStore sets the backingStore property value. Stores model information.
func (m *ItemAddPasswordPostRequestBody) SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore) {
	m.backingStore = value
}

// SetPasswordCredential sets the passwordCredential property value. The passwordCredential property
func (m *ItemAddPasswordPostRequestBody) SetPasswordCredential(value i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable) {
	err := m.GetBackingStore().Set("passwordCredential", value)
	if err != nil {
		panic(err)
	}
}

// ItemAddPasswordPostRequestBodyable
type ItemAddPasswordPostRequestBodyable interface {
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackedModel
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
	GetPasswordCredential() i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable
	SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore)
	SetPasswordCredential(value i6a022527509c6c974d313985d6b1e1814af5796dab5da8f53d13c951e06bb0cd.PasswordCredentialable)
}
