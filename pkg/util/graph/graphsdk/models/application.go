package models

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e "time"

	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
)

// Application
type Application struct {
	DirectoryObject
}

// NewApplication instantiates a new application and sets the default values.
func NewApplication() *Application {
	m := &Application{
		DirectoryObject: *NewDirectoryObject(),
	}
	return m
}

// CreateApplicationFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateApplicationFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	return NewApplication(), nil
}

// GetAppId gets the appId property value. The unique identifier for the application that is assigned to an application by Azure AD. Not nullable. Read-only. Supports $filter (eq).
func (m *Application) GetAppId() *string {
	val, err := m.GetBackingStore().Get("appId")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetCreatedDateTime gets the createdDateTime property value. The date and time the application was registered. The DateTimeOffset type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z. Read-only.  Supports $filter (eq, ne, not, ge, le, in, and eq on null values) and $orderBy.
func (m *Application) GetCreatedDateTime() *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time {
	val, err := m.GetBackingStore().Get("createdDateTime")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time)
	}
	return nil
}

// GetDisplayName gets the displayName property value. The display name for the application. Supports $filter (eq, ne, not, ge, le, in, startsWith, and eq on null values), $search, and $orderBy.
func (m *Application) GetDisplayName() *string {
	val, err := m.GetBackingStore().Get("displayName")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetFieldDeserializers the deserialization information for the current model
func (m *Application) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := m.DirectoryObject.GetFieldDeserializers()
	res["appId"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetAppId(val)
		}
		return nil
	}
	res["createdDateTime"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetTimeValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetCreatedDateTime(val)
		}
		return nil
	}
	res["displayName"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetDisplayName(val)
		}
		return nil
	}
	res["passwordCredentials"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetCollectionOfObjectValues(CreatePasswordCredentialFromDiscriminatorValue)
		if err != nil {
			return err
		}
		if val != nil {
			res := make([]PasswordCredentialable, len(val))
			for i, v := range val {
				if v != nil {
					res[i] = v.(PasswordCredentialable)
				}
			}
			m.SetPasswordCredentials(res)
		}
		return nil
	}
	return res
}

// GetPasswordCredentials gets the passwordCredentials property value. The collection of password credentials associated with the application. Not nullable.
func (m *Application) GetPasswordCredentials() []PasswordCredentialable {
	val, err := m.GetBackingStore().Get("passwordCredentials")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.([]PasswordCredentialable)
	}
	return nil
}

// Serialize serializes information the current object
func (m *Application) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	err := m.DirectoryObject.Serialize(writer)
	if err != nil {
		return err
	}
	{
		err = writer.WriteStringValue("appId", m.GetAppId())
		if err != nil {
			return err
		}
	}
	{
		err = writer.WriteTimeValue("createdDateTime", m.GetCreatedDateTime())
		if err != nil {
			return err
		}
	}
	{
		err = writer.WriteStringValue("displayName", m.GetDisplayName())
		if err != nil {
			return err
		}
	}
	if m.GetPasswordCredentials() != nil {
		cast := make([]i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, len(m.GetPasswordCredentials()))
		for i, v := range m.GetPasswordCredentials() {
			if v != nil {
				cast[i] = v.(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable)
			}
		}
		err = writer.WriteCollectionOfObjectValues("passwordCredentials", cast)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAppId sets the appId property value. The unique identifier for the application that is assigned to an application by Azure AD. Not nullable. Read-only. Supports $filter (eq).
func (m *Application) SetAppId(value *string) {
	err := m.GetBackingStore().Set("appId", value)
	if err != nil {
		panic(err)
	}
}

// SetCreatedDateTime sets the createdDateTime property value. The date and time the application was registered. The DateTimeOffset type represents date and time information using ISO 8601 format and is always in UTC time. For example, midnight UTC on Jan 1, 2014 is 2014-01-01T00:00:00Z. Read-only.  Supports $filter (eq, ne, not, ge, le, in, and eq on null values) and $orderBy.
func (m *Application) SetCreatedDateTime(value *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time) {
	err := m.GetBackingStore().Set("createdDateTime", value)
	if err != nil {
		panic(err)
	}
}

// SetDisplayName sets the displayName property value. The display name for the application. Supports $filter (eq, ne, not, ge, le, in, startsWith, and eq on null values), $search, and $orderBy.
func (m *Application) SetDisplayName(value *string) {
	err := m.GetBackingStore().Set("displayName", value)
	if err != nil {
		panic(err)
	}
}

// SetPasswordCredentials sets the passwordCredentials property value. The collection of password credentials associated with the application. Not nullable.
func (m *Application) SetPasswordCredentials(value []PasswordCredentialable) {
	err := m.GetBackingStore().Set("passwordCredentials", value)
	if err != nil {
		panic(err)
	}
}

// Applicationable
type Applicationable interface {
	DirectoryObjectable
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetAppId() *string
	GetCreatedDateTime() *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time
	GetDisplayName() *string
	GetPasswordCredentials() []PasswordCredentialable
	SetAppId(value *string)
	SetCreatedDateTime(value *i336074805fc853987abe6f7fe3ad97a6a6f3077a16391fec744f671a015fbd7e.Time)
	SetDisplayName(value *string)
	SetPasswordCredentials(value []PasswordCredentialable)
}
