// Code generated by Microsoft Kiota - DO NOT EDIT.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

package models

import (
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e "github.com/microsoft/kiota-abstractions-go/store"
)

type BaseCollectionPaginationCountResponse struct {
	// Stores model information.
	backingStore ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
}

// NewBaseCollectionPaginationCountResponse instantiates a new BaseCollectionPaginationCountResponse and sets the default values.
func NewBaseCollectionPaginationCountResponse() *BaseCollectionPaginationCountResponse {
	m := &BaseCollectionPaginationCountResponse{}
	m.backingStore = ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStoreFactoryInstance()
	return m
}

// CreateBaseCollectionPaginationCountResponseFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
// returns a Parsable when successful
func CreateBaseCollectionPaginationCountResponseFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	return NewBaseCollectionPaginationCountResponse(), nil
}

// GetBackingStore gets the BackingStore property value. Stores model information.
// returns a BackingStore when successful
func (m *BaseCollectionPaginationCountResponse) GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore {
	return m.backingStore
}

// GetFieldDeserializers the deserialization information for the current model
// returns a map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode)(error) when successful
func (m *BaseCollectionPaginationCountResponse) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := make(map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error)
	res["@odata.count"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetInt64Value()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetOdataCount(val)
		}
		return nil
	}
	res["@odata.nextLink"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetOdataNextLink(val)
		}
		return nil
	}
	return res
}

// GetOdataCount gets the @odata.count property value. The OdataCount property
// returns a *int64 when successful
func (m *BaseCollectionPaginationCountResponse) GetOdataCount() *int64 {
	val, err := m.GetBackingStore().Get("odataCount")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*int64)
	}
	return nil
}

// GetOdataNextLink gets the @odata.nextLink property value. The OdataNextLink property
// returns a *string when successful
func (m *BaseCollectionPaginationCountResponse) GetOdataNextLink() *string {
	val, err := m.GetBackingStore().Get("odataNextLink")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// Serialize serializes information the current object
func (m *BaseCollectionPaginationCountResponse) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	{
		err := writer.WriteInt64Value("@odata.count", m.GetOdataCount())
		if err != nil {
			return err
		}
	}
	{
		err := writer.WriteStringValue("@odata.nextLink", m.GetOdataNextLink())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetBackingStore sets the BackingStore property value. Stores model information.
func (m *BaseCollectionPaginationCountResponse) SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore) {
	m.backingStore = value
}

// SetOdataCount sets the @odata.count property value. The OdataCount property
func (m *BaseCollectionPaginationCountResponse) SetOdataCount(value *int64) {
	err := m.GetBackingStore().Set("odataCount", value)
	if err != nil {
		panic(err)
	}
}

// SetOdataNextLink sets the @odata.nextLink property value. The OdataNextLink property
func (m *BaseCollectionPaginationCountResponse) SetOdataNextLink(value *string) {
	err := m.GetBackingStore().Set("odataNextLink", value)
	if err != nil {
		panic(err)
	}
}

type BaseCollectionPaginationCountResponseable interface {
	ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackedModel
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetBackingStore() ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore
	GetOdataCount() *int64
	GetOdataNextLink() *string
	SetBackingStore(value ie8677ce2c7e1b4c22e9c3827ecd078d41185424dd9eeb92b7d971ed2d49a392e.BackingStore)
	SetOdataCount(value *int64)
	SetOdataNextLink(value *string)
}
