package models

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e "time"

	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
)

// DirectoryObject
type DirectoryObject struct {
	Entity
}

// NewDirectoryObject instantiates a new directoryObject and sets the default values.
func NewDirectoryObject() *DirectoryObject {
	m := &DirectoryObject{
		Entity: *NewEntity(),
	}
	return m
}

// CreateDirectoryObjectFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateDirectoryObjectFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
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
				case "#microsoft.graph.servicePrincipal":
					return NewServicePrincipal(), nil
				}
			}
		}
	}
	return NewDirectoryObject(), nil
}

// GetDeletedDateTime gets the deletedDateTime property value. Date and time when this object was deleted. Always null when the object hasn't been deleted.
func (m *DirectoryObject) GetDeletedDateTime() *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time {
	val, err := m.GetBackingStore().Get("deletedDateTime")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time)
	}
	return nil
}

// GetFieldDeserializers the deserialization information for the current model
func (m *DirectoryObject) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := m.Entity.GetFieldDeserializers()
	res["deletedDateTime"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetTimeValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetDeletedDateTime(val)
		}
		return nil
	}
	return res
}

// Serialize serializes information the current object
func (m *DirectoryObject) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	err := m.Entity.Serialize(writer)
	if err != nil {
		return err
	}
	{
		err = writer.WriteTimeValue("deletedDateTime", m.GetDeletedDateTime())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetDeletedDateTime sets the deletedDateTime property value. Date and time when this object was deleted. Always null when the object hasn't been deleted.
func (m *DirectoryObject) SetDeletedDateTime(value *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time) {
	err := m.GetBackingStore().Set("deletedDateTime", value)
	if err != nil {
		panic(err)
	}
}

// DirectoryObjectable
type DirectoryObjectable interface {
	Entityable
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetDeletedDateTime() *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time
	SetDeletedDateTime(value *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time)
}
