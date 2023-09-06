package models

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91 "github.com/microsoft/kiota-abstractions-go/serialization"
)

// ServicePrincipal
type ServicePrincipal struct {
	DirectoryObject
}

// NewServicePrincipal instantiates a new servicePrincipal and sets the default values.
func NewServicePrincipal() *ServicePrincipal {
	m := &ServicePrincipal{
		DirectoryObject: *NewDirectoryObject(),
	}
	odataTypeValue := "#microsoft.graph.servicePrincipal"
	m.SetOdataType(&odataTypeValue)
	return m
}

// CreateServicePrincipalFromDiscriminatorValue creates a new instance of the appropriate class based on discriminator value
func CreateServicePrincipalFromDiscriminatorValue(parseNode i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) (i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable, error) {
	return NewServicePrincipal(), nil
}

// GetAccountEnabled gets the accountEnabled property value. true if the service principal account is enabled; otherwise, false. If set to false, then no users will be able to sign in to this app, even if they are assigned to it. Supports $filter (eq, ne, not, in).
func (m *ServicePrincipal) GetAccountEnabled() *bool {
	val, err := m.GetBackingStore().Get("accountEnabled")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*bool)
	}
	return nil
}

// GetAppId gets the appId property value. The unique identifier for the associated application (its appId property). Supports $filter (eq, ne, not, in, startsWith).
func (m *ServicePrincipal) GetAppId() *string {
	val, err := m.GetBackingStore().Get("appId")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetDisabledByMicrosoftStatus gets the disabledByMicrosoftStatus property value. Specifies whether Microsoft has disabled the registered application. Possible values are: null (default value), NotDisabled, and DisabledDueToViolationOfServicesAgreement (reasons may include suspicious, abusive, or malicious activity, or a violation of the Microsoft Services Agreement).  Supports $filter (eq, ne, not).
func (m *ServicePrincipal) GetDisabledByMicrosoftStatus() *string {
	val, err := m.GetBackingStore().Get("disabledByMicrosoftStatus")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetFieldDeserializers the deserialization information for the current model
func (m *ServicePrincipal) GetFieldDeserializers() map[string]func(i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
	res := m.DirectoryObject.GetFieldDeserializers()
	res["accountEnabled"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetBoolValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetAccountEnabled(val)
		}
		return nil
	}
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
	res["disabledByMicrosoftStatus"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetDisabledByMicrosoftStatus(val)
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
	res["servicePrincipalNames"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetCollectionOfPrimitiveValues("string")
		if err != nil {
			return err
		}
		if val != nil {
			res := make([]string, len(val))
			for i, v := range val {
				if v != nil {
					res[i] = *(v.(*string))
				}
			}
			m.SetServicePrincipalNames(res)
		}
		return nil
	}
	res["servicePrincipalType"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetStringValue()
		if err != nil {
			return err
		}
		if val != nil {
			m.SetServicePrincipalType(val)
		}
		return nil
	}
	res["tags"] = func(n i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.ParseNode) error {
		val, err := n.GetCollectionOfPrimitiveValues("string")
		if err != nil {
			return err
		}
		if val != nil {
			res := make([]string, len(val))
			for i, v := range val {
				if v != nil {
					res[i] = *(v.(*string))
				}
			}
			m.SetTags(res)
		}
		return nil
	}
	return res
}

// GetPasswordCredentials gets the passwordCredentials property value. The collection of password credentials associated with the application. Not nullable.
func (m *ServicePrincipal) GetPasswordCredentials() []PasswordCredentialable {
	val, err := m.GetBackingStore().Get("passwordCredentials")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.([]PasswordCredentialable)
	}
	return nil
}

// GetServicePrincipalNames gets the servicePrincipalNames property value. Contains the list of identifiersUris, copied over from the associated application. Additional values can be added to hybrid applications. These values can be used to identify the permissions exposed by this app within Azure AD. For example,Client apps can specify a resource URI which is based on the values of this property to acquire an access token, which is the URI returned in the 'aud' claim.The any operator is required for filter expressions on multi-valued properties. Not nullable.  Supports $filter (eq, not, ge, le, startsWith).
func (m *ServicePrincipal) GetServicePrincipalNames() []string {
	val, err := m.GetBackingStore().Get("servicePrincipalNames")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.([]string)
	}
	return nil
}

// GetServicePrincipalType gets the servicePrincipalType property value. Identifies whether the service principal represents an application, a managed identity, or a legacy application. This is set by Azure AD internally. The servicePrincipalType property can be set to three different values: Application - A service principal that represents an application or service. The appId property identifies the associated app registration, and matches the appId of an application, possibly from a different tenant. If the associated app registration is missing, tokens are not issued for the service principal.ManagedIdentity - A service principal that represents a managed identity. Service principals representing managed identities can be granted access and permissions, but cannot be updated or modified directly.Legacy - A service principal that represents an app created before app registrations, or through legacy experiences. Legacy service principal can have credentials, service principal names, reply URLs, and other properties which are editable by an authorized user, but does not have an associated app registration. The appId value does not associate the service principal with an app registration. The service principal can only be used in the tenant where it was created.SocialIdp - For internal use.
func (m *ServicePrincipal) GetServicePrincipalType() *string {
	val, err := m.GetBackingStore().Get("servicePrincipalType")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.(*string)
	}
	return nil
}

// GetTags gets the tags property value. Custom strings that can be used to categorize and identify the service principal. Not nullable. The value is the union of strings set here and on the associated application entity's tags property.Supports $filter (eq, not, ge, le, startsWith).
func (m *ServicePrincipal) GetTags() []string {
	val, err := m.GetBackingStore().Get("tags")
	if err != nil {
		panic(err)
	}
	if val != nil {
		return val.([]string)
	}
	return nil
}

// Serialize serializes information the current object
func (m *ServicePrincipal) Serialize(writer i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.SerializationWriter) error {
	err := m.DirectoryObject.Serialize(writer)
	if err != nil {
		return err
	}
	{
		err = writer.WriteBoolValue("accountEnabled", m.GetAccountEnabled())
		if err != nil {
			return err
		}
	}
	{
		err = writer.WriteStringValue("appId", m.GetAppId())
		if err != nil {
			return err
		}
	}
	{
		err = writer.WriteStringValue("disabledByMicrosoftStatus", m.GetDisabledByMicrosoftStatus())
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
	if m.GetServicePrincipalNames() != nil {
		err = writer.WriteCollectionOfStringValues("servicePrincipalNames", m.GetServicePrincipalNames())
		if err != nil {
			return err
		}
	}
	{
		err = writer.WriteStringValue("servicePrincipalType", m.GetServicePrincipalType())
		if err != nil {
			return err
		}
	}
	if m.GetTags() != nil {
		err = writer.WriteCollectionOfStringValues("tags", m.GetTags())
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAccountEnabled sets the accountEnabled property value. true if the service principal account is enabled; otherwise, false. If set to false, then no users will be able to sign in to this app, even if they are assigned to it. Supports $filter (eq, ne, not, in).
func (m *ServicePrincipal) SetAccountEnabled(value *bool) {
	err := m.GetBackingStore().Set("accountEnabled", value)
	if err != nil {
		panic(err)
	}
}

// SetAppId sets the appId property value. The unique identifier for the associated application (its appId property). Supports $filter (eq, ne, not, in, startsWith).
func (m *ServicePrincipal) SetAppId(value *string) {
	err := m.GetBackingStore().Set("appId", value)
	if err != nil {
		panic(err)
	}
}

// SetDisabledByMicrosoftStatus sets the disabledByMicrosoftStatus property value. Specifies whether Microsoft has disabled the registered application. Possible values are: null (default value), NotDisabled, and DisabledDueToViolationOfServicesAgreement (reasons may include suspicious, abusive, or malicious activity, or a violation of the Microsoft Services Agreement).  Supports $filter (eq, ne, not).
func (m *ServicePrincipal) SetDisabledByMicrosoftStatus(value *string) {
	err := m.GetBackingStore().Set("disabledByMicrosoftStatus", value)
	if err != nil {
		panic(err)
	}
}

// SetPasswordCredentials sets the passwordCredentials property value. The collection of password credentials associated with the application. Not nullable.
func (m *ServicePrincipal) SetPasswordCredentials(value []PasswordCredentialable) {
	err := m.GetBackingStore().Set("passwordCredentials", value)
	if err != nil {
		panic(err)
	}
}

// SetServicePrincipalNames sets the servicePrincipalNames property value. Contains the list of identifiersUris, copied over from the associated application. Additional values can be added to hybrid applications. These values can be used to identify the permissions exposed by this app within Azure AD. For example,Client apps can specify a resource URI which is based on the values of this property to acquire an access token, which is the URI returned in the 'aud' claim.The any operator is required for filter expressions on multi-valued properties. Not nullable.  Supports $filter (eq, not, ge, le, startsWith).
func (m *ServicePrincipal) SetServicePrincipalNames(value []string) {
	err := m.GetBackingStore().Set("servicePrincipalNames", value)
	if err != nil {
		panic(err)
	}
}

// SetServicePrincipalType sets the servicePrincipalType property value. Identifies whether the service principal represents an application, a managed identity, or a legacy application. This is set by Azure AD internally. The servicePrincipalType property can be set to three different values: Application - A service principal that represents an application or service. The appId property identifies the associated app registration, and matches the appId of an application, possibly from a different tenant. If the associated app registration is missing, tokens are not issued for the service principal.ManagedIdentity - A service principal that represents a managed identity. Service principals representing managed identities can be granted access and permissions, but cannot be updated or modified directly.Legacy - A service principal that represents an app created before app registrations, or through legacy experiences. Legacy service principal can have credentials, service principal names, reply URLs, and other properties which are editable by an authorized user, but does not have an associated app registration. The appId value does not associate the service principal with an app registration. The service principal can only be used in the tenant where it was created.SocialIdp - For internal use.
func (m *ServicePrincipal) SetServicePrincipalType(value *string) {
	err := m.GetBackingStore().Set("servicePrincipalType", value)
	if err != nil {
		panic(err)
	}
}

// SetTags sets the tags property value. Custom strings that can be used to categorize and identify the service principal. Not nullable. The value is the union of strings set here and on the associated application entity's tags property.Supports $filter (eq, not, ge, le, startsWith).
func (m *ServicePrincipal) SetTags(value []string) {
	err := m.GetBackingStore().Set("tags", value)
	if err != nil {
		panic(err)
	}
}

// ServicePrincipalable
type ServicePrincipalable interface {
	DirectoryObjectable
	i878a80d2330e89d26896388a3f487eef27b0a0e6c010c493bf80be1452208f91.Parsable
	GetAccountEnabled() *bool
	GetAppId() *string
	GetDisabledByMicrosoftStatus() *string
	GetPasswordCredentials() []PasswordCredentialable
	GetServicePrincipalNames() []string
	GetServicePrincipalType() *string
	GetTags() []string
	SetAccountEnabled(value *bool)
	SetAppId(value *string)
	SetDisabledByMicrosoftStatus(value *string)
	SetPasswordCredentials(value []PasswordCredentialable)
	SetServicePrincipalNames(value []string)
	SetServicePrincipalType(value *string)
	SetTags(value []string)
}
