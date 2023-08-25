package models

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e "github.com/microsoft/kiota-abstractions-go/store"
)

// Entity
type Entity struct {
	// Stores model information.
	backingStore ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
}

// NewEntity instantiates a new entity and sets the default values.
func NewEntity() *Entity {
	m := &Entity{}
	m.backingStore = ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStoreFactoryInstance()
	return m
}

// CreateEntityFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateEntityFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	if parseNode != nil {
		mappingValueNode, err := parseNode.GetChildNode("@odata.type")
		if err != nil {
			return nil, err
		}
		if mappingValueNode != nil {
			mappingValue, err := mappingValueNode.GetStringValue()
			if err != nil {
				return nil, err
			}
			if mappingValue != nil {
				switch *mappingValue {
				case "#microsoft.graph.application":
					return NewApplication(), nil
				case "#microsoft.graph.directoryObject":
					return NewDirectoryObject(), nil
				case "#microsoft.graph.servicePrincipal":
					return NewServicePrincipal(), nil
				}
			}
		}
	}
	return NewEntity(), nil
}

// GetBackingStore gets the backingStore property value. Stores model information.
func (m *Entity) GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore {
	return m.backingStore
}

// GetFieldDeserializers the deserialization information for the current model
func (m *Entity) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := make(map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error)
	res["id"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetId(val)
		}
		return nil
	}
	res["@odata.type"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetOdataType(val)
		}
		return nil
	}
	return res
}

// GetId gets the id property value. The unique idenfier for an entity. Read-only.
func (m *Entity) GetId() *string {
	val, err := m.GetBackingStore().Get("id")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetOdataType gets the @odata.type property value. The OdataType property
func (m *Entity) GetOdataType() *string {
	val, err := m.GetBackingStore().Get("odataType")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// Serialize serializes information the current object
func (m *Entity) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	{
		err := writer.WriteStringValue("id", m.GetId())
		if err != nil {
			return err
		}
	}
	{
		err := writer.WriteStringValue("@odata.type", m.GetOdataType())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBackingStore sets the backingStore property value. Stores model information.
func (m *Entity) SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore) {
	m.backingStore = value
}

// SetId sets the id property value. The unique idenfier for an entity. Read-only.
func (m *Entity) SetId(value *string) {
	err := m.GetBackingStore().Set("id", value)
	if err != nil {
		panic(err)
	}
}

// SetOdataType sets the @odata.type property value. The OdataType property
func (m *Entity) SetOdataType(value *string) {
	err := m.GetBackingStore().Set("odataType", value)
	if err != nil {
		panic(err)
	}
}

// Entityable
type Entityable interface {
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackedModel
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
	GetId() *string
	GetOdataType() *string
	SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore)
	SetId(value *string)
	SetOdataType(value *string)
}
