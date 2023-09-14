package odataerrors

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e "github.com/microsoft/kiota-abstractions-go/store"
)

// ErrorDetails
type ErrorDetails struct {
	// Stores model information.
	backingStore ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
}

// NewErrorDetails instantiates a new ErrorDetails and sets the default values.
func NewErrorDetails() *ErrorDetails {
	m := &ErrorDetails{}
	m.backingStore = ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStoreFactoryInstance()
	return m
}

// CreateErrorDetailsFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateErrorDetailsFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	return NewErrorDetails(), nil
}

// GetBackingStore gets the backingStore property value. Stores model information.
func (m *ErrorDetails) GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore {
	return m.backingStore
}

// GetCode gets the code property value. The code property
func (m *ErrorDetails) GetCode() *string {
	val, err := m.GetBackingStore().Get("code")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetFieldDeserializers the deserialization information for the current model
func (m *ErrorDetails) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := make(map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error)
	res["code"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetCode(val)
		}
		return nil
	}
	res["message"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetMessage(val)
		}
		return nil
	}
	res["target"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetTarget(val)
		}
		return nil
	}
	return res
}

// GetMessage gets the message property value. The message property
func (m *ErrorDetails) GetMessage() *string {
	val, err := m.GetBackingStore().Get("message")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetTarget gets the target property value. The target property
func (m *ErrorDetails) GetTarget() *string {
	val, err := m.GetBackingStore().Get("target")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// Serialize serializes information the current object
func (m *ErrorDetails) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	{
		err := writer.WriteStringValue("code", m.GetCode())
		if err != nil {
			return err
		}
	}
	{
		err := writer.WriteStringValue("message", m.GetMessage())
		if err != nil {
			return err
		}
	}
	{
		err := writer.WriteStringValue("target", m.GetTarget())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBackingStore sets the backingStore property value. Stores model information.
func (m *ErrorDetails) SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore) {
	m.backingStore = value
}

// SetCode sets the code property value. The code property
func (m *ErrorDetails) SetCode(value *string) {
	err := m.GetBackingStore().Set("code", value)
	if err != nil {
		panic(err)
	}
}

// SetMessage sets the message property value. The message property
func (m *ErrorDetails) SetMessage(value *string) {
	err := m.GetBackingStore().Set("message", value)
	if err != nil {
		panic(err)
	}
}

// SetTarget sets the target property value. The target property
func (m *ErrorDetails) SetTarget(value *string) {
	err := m.GetBackingStore().Set("target", value)
	if err != nil {
		panic(err)
	}
}

// ErrorDetailsable
type ErrorDetailsable interface {
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackedModel
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
	GetCode() *string
	GetMessage() *string
	GetTarget() *string
	SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore)
	SetCode(value *string)
	SetMessage(value *string)
	SetTarget(value *string)
}
