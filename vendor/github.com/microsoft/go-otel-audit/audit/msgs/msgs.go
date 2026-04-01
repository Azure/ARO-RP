/*
Package msgs contains the types and constants for audit messages.

These are based on https://msazure.visualstudio.com/One/_git/ASM-OpenTelemetryAudit?path=/src/csharp/OpenTelemetry.Audit.Geneva/DataModel

The main message type is the Msg. It acts as a wrapper around all message types.

Users send two different message types, DataPlane and ControlPlane. Each of these types uses a Record
sub-message to send the actual data. DataPlane and ControlPlane simply signify what type of system is
logging the message.

The main type that users interact with is the Record type. This details all the security audit information. You can
construct a Record using either a composite literal or the New() function. The New() function will allocate maps for
you, but the composite literal will not.

The most efficient way to create an AuditRecord is to build a package around AuditRecords that have static maps for various
use cases.  Then have functions that assemble the AuditRecord from these into an AuditRecord to avoid any allocations.

To help with building having base Record that you can use to fill out details, we provide a Clone() method on Record.
This will do a deep clone of the Record. This is useful when you have a base Record that you want to use as a
template for other Record objects. However, check for the limitations on custom data for cloning.
*/
package msgs

import (
	"bytes"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"
	"unsafe"

	"github.com/go-json-experiment/json"
	"github.com/vmihailenco/msgpack/v4"
	"golang.org/x/exp/constraints"
)

// now is a function that returns the current time. This is used to allow for testing.
type now func() time.Time

//go:generate stringer -type=Type -linecomment

// Type is the type of audit record, either DataPlane or ControlPlane.
type Type uint8

func (a Type) private() {}

// Note: Do not change the order, as this is used in validation.
const (
	// ATUnknown is an unknown audit type. This should never be used and indicates
	// a bug in the code
	ATUnknown Type = 0
	// DataPlane indicates this type of audit record is for the data plane.
	DataPlane Type = 1 // AsmAuditDP
	// ControlPlane indicates this type of audit record is for the control plane.
	ControlPlane Type = 2 // AsmAuditCP
	// Heartbeat indicates this type of audit record is for the heartbeat. This is not
	// usable by a user.
	Heartbeat Type = 3 // AsmAuditHB
	// Diagnostic indicates this type of audit record is for diagnostic purposes when a
	// message fails. This is not usable by a user.
	Diagnostic Type = 4 // AsmAuditDG
)

// Msg is a message sent to the audit server. This wraps all message types.
// Users should only set the Type to either DataPlane or ControlPlane and the
// corresponding Record field.
type Msg struct {
	now now

	// Type is the type of message.
	Type Type

	// Record is an audit record.
	Record Record
	// Heartbeat is a heartbeat message.
	Heartbeat HeartbeatMsg
	// Diagnostic is a diagnostic message.
	Diagnostic DiagnosticMsg
}

// MarshalMsgpack marshals the AuditRecord to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
func MarshalMsgpack(msg Msg) ([]byte, error) {
	a, err := marshalPrep(msg)
	if err != nil {
		return nil, err
	}

	return msgpack.Marshal(a)
}

type timeFormat struct {
	TimeFormat string
}

var genevaTF = &timeFormat{TimeFormat: "DateTime"}

// marshalPrep is a helper function that prepares the AuditRecord for marshaling.
// The message has a funky structure around it that is required by the Geneva Agent.
// This also does some compaction on slices that might have duplicate values, where duplicates
// are not allowed. This allows us to remove duplicates and have an ordered slice.
// We also use arrays instead of slices for the outer structure to avoid allocations.
// Note: In case you are thinking that slices are going perform better here, they won't. Its more than 3
// times the allocation cost to use slices here and almost twice as long to prep.
// BenchmarkMarshalPrep-10    	 3129526	       383.2 ns/op	     464 B/op	      12 allocs/op (slices)
// BenchmarkMarshalPrep-10    	 6631240	       179.0 ns/op	     264 B/op	       4 allocs/op
// Note: To get this down further, we'd need to build a msgpack encoder that can encode directly to a buffer and then
// skip the marshal steps. Then we could get this part down to a zero allocation cost. The problem is that
// the "any" type creates an allocation in many cases. Not sure its worth it. All message marshalling
// and unmarshalling is marked as internal, so we can change it later if we need to.
func marshalPrep(msg Msg) ([3]any, error) {
	n := msg.now
	if msg.now == nil {
		n = time.Now
	}

	var a any
	switch msg.Type {
	case DataPlane, ControlPlane:
		// These slices should only have unique values. These are small, should be cheap.
		slices.Sort(msg.Record.OperationCategories)
		msg.Record.OperationCategories = slices.Compact(msg.Record.OperationCategories)
		slices.Sort(msg.Record.CallerAccessLevels)
		msg.Record.CallerAccessLevels = slices.Compact(msg.Record.CallerAccessLevels)
		a = msg.Record
	case Heartbeat:
		a = msg.Heartbeat
	case Diagnostic:
		a = msg.Diagnostic
	default:
		return [3]any{}, fmt.Errorf("unknown audit type: %d", msg.Type)
	}

	return [3]any{
		msg.Type.String(),
		[1][2]any{
			[2]any{ // If the IDE says this is redundant, it is. Leave it alone anyways.
				n().Unix(),
				a,
			},
		},
		genevaTF, // required by Geneva Agent
	}, nil
}

// Addr represents an IP address. It is a wrapper around netip.Addr to allow
// for MessagePack serialization.
type Addr struct {
	netip.Addr
}

// ParseAddr parses a string into an Addr.
func ParseAddr(s string) (Addr, error) {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return Addr{}, err
	}
	return Addr{Addr: ip}, nil
}

// MustParseAddr parses a string into an Addr. It panics if there is an error.
// This is usually useful for testing and not services.
func MustParseAddr(s string) Addr {
	a, err := ParseAddr(s)
	if err != nil {
		panic(err)
	}
	return a
}

// NetIP returns the netip.Addr of the Addr.
func (a Addr) NetIP() netip.Addr {
	return a.Addr
}

// MarshalMsgpack marshals the Addr to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
func (a Addr) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(a.String())
}

// UnmarshalMsgpack unmarshals the Addr from MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
func (a *Addr) UnmarshalMsgpack(b []byte) error {
	var s string
	if err := msgpack.Unmarshal(b, &s); err != nil {
		return err
	}
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return err
	}
	a.Addr = ip
	return nil
}

// UnmarshalJSON unmarshals the value from JSON.
// This is reserved for internal use only (aka no semantic version promises).
func (a *Addr) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return err
	}
	a.Addr = ip
	return nil
}

//go:generate stringer -type=OperationType

// OperationType represents the type of audit operation that is being recorded.
type OperationType uint16 // Do not use uint8, as this gets used in a slice, which can get confused as []byte.

// MarshalMsgpack marshals the OperationType to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
// Note: DO NOT MAKE THIS A POINTER RECEIVER. There is a bug in v4 msgpack that thinks this
// is non-addressable, therefore it can't be used. So this needs to stay as a value receiver.
// We don't use v5 because it has some other bugs that are not fixed yet.
func (o OperationType) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(o.String())
}

// UnmarshalMsgpack unmarshals the OperationType from MessagePack format.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationType) UnmarshalMsgpack(b []byte) error {
	// Note: we drop the first byte of a MsgPack entry because it denotes the MsgPack type.
	// We already know the type, so we don't need to do so.
	s := bytesToStr(b[1:])
	*o = unString[uint8, OperationType](s, _OperationType_name, _OperationType_index[:])
	return nil
}

// UnmarshalJSON unmarshals the value from JSON.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationType) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"'`)
	s := bytesToStr(b)
	*o = unString[uint8, OperationType](s, _OperationType_name, _OperationType_index[:])
	return nil
}

const (
	// UnknownOperationType represents an unknown operation type. This is indication of a bug.
	UnknownOperationType OperationType = 0
	// Read represents a read operation.
	Read OperationType = 1
	// Update represents an update operation.
	Update OperationType = 2
	// Create represents a create operation.
	Create OperationType = 3
	// Delete represents a delete operation.
	Delete OperationType = 4
)

//go:generate stringer -type=OperationCategory

// OperationCategory represents information about the category of the operation being audited.
// An operation can be in multiple categories.
type OperationCategory uint16

// MarshalMsgpack marshals the OperationCategory to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
// Note: DO NOT MAKE THIS A POINTER RECEIVER. There is a bug in v4 msgpack that thinks this
// is non-addressable, therefore it can't be used. So this needs to stay as a value receiver.
// We don't use v5 because it has some other bugs that are not fixed yet.
func (o OperationCategory) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(o.String())
}

// UnmarshalMsgpack unmarshals the OperationCategory from MessagePack format.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationCategory) UnmarshalMsgpack(b []byte) error {
	s := bytesToStr(b[1:])
	*o = unString[uint16, OperationCategory](s, _OperationCategory_name, _OperationCategory_index[:])
	return nil
}

// UnmarshalJSON unmarshals the value from JSON.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationCategory) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"'`)
	s := bytesToStr(b)
	*o = unString[uint16, OperationCategory](s, _OperationCategory_name, _OperationCategory_index[:])
	return nil
}

const (
	// UnknownOperationCategory represents an unknown operation category. This is indication of a bug.
	UnknownOperationCategory OperationCategory = 0
	// OCOther is an operation category not listed here.
	OCOther OperationCategory = 1
	// UserManagement indicates the operation is related to user management.
	// These are operations such as "create AAD user" or delete system user".
	UserManagement OperationCategory = 2
	// GroupManagement indicates the operation is related to group management.
	// These are operations like "add a user to a group" or "delete group".
	GroupManagement OperationCategory = 3
	// Authentication indicates the operation is related to authentication.
	// "AAD authentication" would be an example of this.
	Authentication OperationCategory = 4
	// Authorization indicates the operation is related to authorization.
	// "OAuth authorization" would be an example of this.
	Authorization OperationCategory = 5
	// RoleManagement indicates the operation is related to role management.
	// "Changing a user's permissions" would be an example of this.
	RoleManagement OperationCategory = 6
	// ApplicationManagement indicates the operation is related to application management.
	// "Registering an application in AAD" or "deleting an application in AAD" would be examples of this.
	ApplicationManagement OperationCategory = 7
	// KeyManagement indicates the operation is related to key management.
	// Read/Create/Delete key in KeyVault would be an example of this.
	KeyManagement OperationCategory = 8
	// DirectoryManagement indicates the operation is related to directory management.
	// "Create a new directory in a filesystem" or "update Azure Storage directory metadata" would be examples of this.
	DirectoryManagement OperationCategory = 9
	// ResourceManagement indicates the operation is related to resource management.
	// Any CRUD operation on a resource would be an example of this.
	ResourceManagement OperationCategory = 10
	// PolicyManagement indicates the operation is related to policy management.
	// "Update system policy" would be an example of this.
	PolicyManagement OperationCategory = 11
	// DeviceManagement indicates the operation is related to device management.
	// TODO(jdoak): This is not in the documentation, so need an example.
	DeviceManagement OperationCategory = 12
	// EntitlementManagement indicates the operation is related to entitlement management.
	// "Create an access pacakge in AAD" would be an example of this.
	EntitlementManagement OperationCategory = 13
	// PasswordManagement indicates the operation is related to password management.
	// "Reset a user's password" would be an example of this.
	PasswordManagement OperationCategory = 14
	// IdentityProtection indicates the operation is related to identity protection.
	// "AAD - Create a conditional access policy to allow users to self-mediate" would be an example of this.
	IdentityProtection OperationCategory = 15
	// ObjectManagement indicates the operation is related to object management.
	// This is an abstract category, can be used for any object type.
	ObjectManagement OperationCategory = 16
	// ProvisioningManagement indicates the operation is related to provisioning management.
	// "Create a provisioning job based on a template" would be an example of this.
	ProvisioningManagement OperationCategory = 17
	// CustomerFacing indicates the operation is related to customer facing.
	// This is a required category for Activity Log surfaced audit logs.
	// TODO(jdoak): This sounds Windows specific, I need more information here.
	CustomerFacing OperationCategory = 18
	// EUII indicates the operation is related to end user identifiable information (alias, email).
	// This category means that CallerIdentity field will be obfuscated.
	EUII OperationCategory = 19
)

//go:generate stringer -type=OperationResult

// OperationResult represents the outcome of the operation.
type OperationResult uint16

// MarshalMsgpack marshals the OperationResult to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
// Note: DO NOT MAKE THIS A POINTER RECEIVER. There is a bug in v4 msgpack that thinks this
// is non-addressable, therefore it can't be used. So this needs to stay as a value receiver.
// We don't use v5 because it has some other bugs that are not fixed yet.
func (o OperationResult) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(o.String())
}

// UnmarshalMsgpack unmarshals OperationResult the from MessagePack format.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationResult) UnmarshalMsgpack(b []byte) error {
	s := bytesToStr(b[1:])
	*o = unString[uint8, OperationResult](s, _OperationResult_name, _OperationResult_index[:])
	return nil
}

// UnmarshalJSON unmarshals the value from JSON.
// This is reserved for internal use only (aka no semantic version promises).
func (o *OperationResult) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"'`)
	s := bytesToStr(b)
	*o = unString[uint8, OperationResult](s, _OperationResult_name, _OperationResult_index[:])
	return nil
}

const (
	// UnknownOperationResult represents an unknown operation result. This is indication of a bug.
	UnknownOperationResult OperationResult = 0
	// Success represents a successful operation.
	Success OperationResult = 1
	// Failure represents a failed operation.
	Failure OperationResult = 2
)

//go:generate stringer -type=CallerIdentityType

// CallerIdentityType represents the identity types.
type CallerIdentityType uint16

// MarshalMsgpack marshals the OperationResult to MessagePack.
// This is reserved for internal use only (aka no semantic version promises).
// Note: DO NOT MAKE THIS A POINTER RECEIVER. There is a bug in v4 msgpack that thinks this
// is non-addressable, therefore it can't be used. So this needs to stay as a value receiver.
// We don't use v5 because it has some other bugs that are not fixed yet.
func (c CallerIdentityType) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(c.String())
}

// UnmarshalMsgpack unmarshals CallerIdentityType the from MessagePack format.
// This is reserved for internal use only (aka no semantic version promises).
func (o *CallerIdentityType) UnmarshalMsgpack(b []byte) error {
	s := bytesToStr(b[1:])
	*o = unString[uint8, CallerIdentityType](s, _CallerIdentityType_name, _CallerIdentityType_index[:])
	return nil
}

// UnmarshalJSON unmarshals the value from JSON.
// This is reserved for internal use only (aka no semantic version promises).
func (c *CallerIdentityType) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"'`)
	s := bytesToStr(b)
	*c = unString[uint8, CallerIdentityType](s, _CallerIdentityType_name, _CallerIdentityType_index[:])
	return nil
}

const (
	// UnknownCallerIdentityType represents an unknown caller identity type. This is indication of a bug.
	UnknownCallerIdentityType CallerIdentityType = 0
	// CIOther is an identity type not listed here.
	CIOther CallerIdentityType = 1
	// UPN is a User Principal Name, like	"username@domain".
	UPN CallerIdentityType = 2
	// PUID is a Personal User ID, which will be a GUID in string format.
	PUID CallerIdentityType = 3
	// ObjectID is an Object identifier, which will be a GUID in string format.
	ObjectID CallerIdentityType = 4
	// Certificate is a Certificate information. Must have at least one of the following: Thumbprint, Subject or Issuer.
	// Common format: "Thumbprint:40_alphanumeric_character, Subject:CN=Common_Name, Issuer:CN=Common_Name, OU=Organizational_Unit_name, O=Organization_name, L=Locality_name, S=State_or_Province_name>, C=Country"
	Certificate CallerIdentityType = 5
	// Claim is claim information. Identity:alphanumeric_characters_and_dashes-Claims:URI_list_of_claims_separated_by_semicolons
	Claim CallerIdentityType = 6
	// Username is a User name. DOMAIN\ALIAS or MACHINENAME\ACCOUNTNAME.
	Username CallerIdentityType = 7
	// AccessKeyName is Access key information. Can be any format, depending on key type.
	AccessKeyName CallerIdentityType = 8
	// SubscriptionID is a Subsription identifier, which will be a GUID in string format.
	SubscriptionID CallerIdentityType = 9
	// ApplicationID is an Application identifier, which will be a GUID in string format.
	ApplicationID CallerIdentityType = 10
	// TenantID is a Tenant identifier, which will be a GUID in string format.
	TenantID CallerIdentityType = 11
	// TokenID is a Token identifier, which will be a GUID in string format.
	TokenID CallerIdentityType = 12
	// SASUrl is a Shared Access Signature URL. It is important to remove SAS token part from URL.
	SasURL CallerIdentityType = 13
	// TODO(jdoak): Get a description for this, it wasn't in the documentation.
	System CallerIdentityType = 14
)

// CallerIdentityEntry represents a caller identity entry.
type CallerIdentityEntry struct {
	// Identity is the identity in string format. Example: "audituser@microsoft.com".
	Identity string
	// Description is a description of the identity.
	Description string
}

// Validate runs validation rules over the CallerIdentityEntry.
func (c CallerIdentityEntry) Validate() error {
	if c.Identity == "" {
		return fmt.Errorf("identity is required")
	}
	if c.Description == "" {
		return fmt.Errorf("description is required")
	}

	return nil
}

// TargetResourceEntry represents a target resource entry.
type TargetResourceEntry struct {
	// Name is the name of the target resource. Example: "/subscriptions/4bb12743-2418-427f-92ae-96a11a4044fe/resourceGroups/IFxAudit/providers/Microsoft.Storage/storageAccounts/myaccount"
	// (Required).
	Name string
	// Cluster is the name of the cluster the target is in. (Optional).
	Cluster string
	// DataCenter is the name of the data center the target is in. (Optional).
	DataCenter string
	// Region is the name of the region the target is in. (Optional).
	Region string
}

// Validate runs validation rules over the TargetResourceEntry.
func (t TargetResourceEntry) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("target resource name is required")
	}

	return nil
}

// Record represents an audit record.
// Note: The fields in this struct are ordered for field alignment to reduce size due to padding.
// As we can have lots of these records in memory at any one time, the field alignment is important
// for reducing memory usage.
type Record struct {
	// Hook is a hook that is called before sending the AuditRecord to the server.
	// You can use this do introspection on the AuditRecord before it is sent to the server and do things
	// like update metric.
	// If the hook returns an error, the error will be logged. However, the AuditRecord will still be sent to the server.
	// Be careful with this, because if you make Hook blocking, it blocks the sending of the AuditRecord to the server until
	// this completes. Any modifications to the AuditRecord are returned to the caller.  If you modify but ignore the
	// returned AuditRecord, then only modifications to reference type data will be reflected in the AuditRecord sent to
	// the server.
	Hook func(a Record) (Record, error) `msgpack:"-" json:"-"`
	now  func() time.Time               `msgpack:"-" json:"-"`

	// CallerIpAddress is the IP address of the caller.
	CallerIpAddress Addr
	// CallerIdentities is the identities that initiated the operation.
	CallerIdentities map[CallerIdentityType][]CallerIdentityEntry
	// OperationCategories is a collection of all audit categories that apply to this audit event.
	OperationCategories []OperationCategory
	// CustomData is a dictionary of string-object key-value pairs that can be used to extend the audit record.
	CustomData map[string]any `msgpack:",omitempty" json:",omitempty"`
	// TargetResources is a collection of target resources that got affected by Operation.
	// Keys are the resource that got affected, here are few examples:
	//
	// - Tenant
	// - ManagementGroup
	// - ManagementGroupId
	// - SubscriptionId
	// - SubscriptionName
	// - ServiceGroupOid
	// - TeamGroupId
	//
	// The Name in the TargetResourceEntry is the resource that got affected, this is usually
	// the UUID of the resource.
	TargetResources map[string][]TargetResourceEntry
	// CallerAccessLevels is the RBAC granted to user to execute current operation.
	CallerAccessLevels []string
	// OperationAccessLevel is the minimum access level (role) required for the operation to be performed.
	OperationAccessLevel string
	// OperationName is the name of the operation being audited.
	OperationName string
	// OperationResultDescription provides specific details for the operation result. These details can include tracing information,
	// such as the symptoms, of the result that are used for further analysis (ex: exception message).
	OperationResultDescription string
	// CallerAgent is used to identify what tool or system this operation
	// is being executed from. This will allow an official inventory of Azure-wide
	// tooling, compliant/non-compliant tooling, approved deployment systems, etc.
	CallerAgent string
	// OperationCategoryDescription provides specific details for the operation category.
	// This field is required if OperationCategories contains "Other" item.
	OperationCategoryDescription string
	// OperationType is the type of the operation.
	OperationType OperationType
	// OperationResult is the outcome of the operation.
	OperationResult OperationResult
}

// New is the constructor for a Record. This simply initializes the Msg as type "t", which must be
// a DataPlane or ControlPlane type. It also initializes all maps to prevent nil use errors.
// However, it is recommended that if you can create a complete AuditRecord in one go, you use that
// instead of this constructor to avoid any unnecessary allocations of maps you don't need.
func New(t Type) (Msg, error) {
	switch t {
	case DataPlane, ControlPlane:
	default:
		return Msg{}, fmt.Errorf("invalid audit record type for New(): %v", t)
	}

	return Msg{
		Type: t,
		Record: Record{
			CallerIdentities: map[CallerIdentityType][]CallerIdentityEntry{},
			TargetResources:  map[string][]TargetResourceEntry{},
			CustomData:       map[string]any{},
		},
	}, nil
}

// Validate runs validation rules over the AuditRecord.
// Taken from: https://msazure.visualstudio.com/One/_git/ASM-OpenTelemetryAudit?path=/src/csharp/OpenTelemetry.Audit.Geneva/AuditRecordValidator.cs&version=GBmain&_a=contents
func (a Record) Validate() (err error) {
	// Note: We don't do validation that something like an OperationCategory is valid, as that is
	// a custom type.  It is possible for someone to do like OperationCategory(1000),
	// but unlikely. Unless the server doesn't do any validation, this should be fine.
	// If the server doesn't do any validation, then it should be fixed there if this is a problem.
	// Because anyone can just write to the security endpoint.

	if a.OperationName == "" {
		return fmt.Errorf("operation name is required")
	}

	if len(a.OperationCategories) == 0 {
		return fmt.Errorf("at least one operation category is required")
	}

	for _, category := range a.OperationCategories {
		if category == OCOther && a.OperationCategoryDescription == "" {
			return fmt.Errorf("operation category description is required for category %v", OCOther)
		}
	}

	if a.OperationResult == Failure && a.OperationResultDescription == "" {
		return fmt.Errorf("operation result description is required for failed operations")
	}

	if a.OperationAccessLevel == "" {
		return fmt.Errorf("operation access level is required")
	}

	if a.CallerAgent == "" {
		return fmt.Errorf("caller agent is required")
	}

	if len(a.CallerIdentities) == 0 {
		return fmt.Errorf("at least one caller identity is required")
	}

	for identityType, identities := range a.CallerIdentities {
		if len(identities) == 0 {
			return fmt.Errorf("at least one identity is required for identity type %v", identityType)
		}
		for _, identity := range identities {
			if err := identity.Validate(); err != nil {
				return err
			}
		}
	}

	if !a.CallerIpAddress.IsValid() {
		return fmt.Errorf("caller IP address is required")
	}
	if a.CallerIpAddress.IsUnspecified() {
		return fmt.Errorf("caller IP address cannot be unspecified")
	}
	if a.CallerIpAddress.IsLoopback() {
		return fmt.Errorf("caller IP address cannot be loopback")
	}
	if a.CallerIpAddress.IsMulticast() {
		return fmt.Errorf("caller IP address cannot be multicast")
	}

	if len(a.CallerAccessLevels) == 0 {
		return fmt.Errorf("at least one caller access level is required")
	}

	for _, k := range a.CallerAccessLevels {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("caller access level cannot be empty")
		}
	}

	if len(a.TargetResources) == 0 {
		return fmt.Errorf("at least one target resource is required")
	}

	for resourceType, resources := range a.TargetResources {
		if strings.TrimSpace(resourceType) == "" {
			return fmt.Errorf("target resource type cannot be empty")
		}
		for _, resource := range resources {
			if err := resource.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Clone returns a clone of the type it is attached to.
type Cloner interface {
	Clone() any
}

// Clone returns a clone of the AuditRecord.
// Note: This is a deep clone, with the exception of CustomData. CustomData has its keys cloned,
// but the values may not be cloned if the data is a reference type, pointer or struct with those types.
// If the value implements Cloner, then that method will be called. If it doesn't, then it will be copied
// by value. If you need to clone a reference type, pointer or struct with those types, then you must
// implement Cloner on that type.
func (a Record) Clone() Record {
	a.CallerIdentities = cloneCallerIdentities(a.CallerIdentities)
	a.OperationCategories = slices.Clone(a.OperationCategories)
	a.TargetResources = cloneTargetResources(a.TargetResources)
	a.CallerAccessLevels = slices.Clone(a.CallerAccessLevels)
	a.CustomData = cloneCustomData(a.CustomData)

	return a
}

func cloneCallerIdentities(callerIdentities map[CallerIdentityType][]CallerIdentityEntry) map[CallerIdentityType][]CallerIdentityEntry {
	if callerIdentities == nil {
		return nil
	}
	clone := make(map[CallerIdentityType][]CallerIdentityEntry, len(callerIdentities))
	for k, entries := range callerIdentities {
		clone[k] = make([]CallerIdentityEntry, len(entries))
		copy(clone[k], entries)
	}
	return clone
}

func cloneTargetResources(targetResources map[string][]TargetResourceEntry) map[string][]TargetResourceEntry {
	if targetResources == nil {
		return nil
	}
	clone := make(map[string][]TargetResourceEntry, len(targetResources))
	for k, entries := range targetResources {
		clone[k] = make([]TargetResourceEntry, len(entries))
		copy(clone[k], entries)
	}
	return clone
}

func cloneCustomData(customData map[string]any) map[string]any {
	if customData == nil {
		return nil
	}
	clone := make(map[string]any, len(customData))
	for k, v := range customData {
		if cloner, ok := v.(Cloner); ok {
			clone[k] = cloner.Clone()
		} else {
			clone[k] = v
		}
	}
	return clone
}

type uintConstraint interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// unString is a helper function to convert a string representation done by the stringer tool
// back into the original type.
func unString[I constraints.Unsigned, R uintConstraint](from, names string, index []I) R {
	from = strings.ToLower(from)
	l := strings.ToLower(names)
	for i := 0; i < len(index)-1; i++ {
		if from == l[index[i]:index[i+1]] {
			return R(i)
		}
	}
	return R(0)
}

// bytesToStr converts a byte slice to a string without a copy (aka no allocation).
// This uses unsafe, so only use in cases where you know the byte slice is not going to be modified.
func bytesToStr(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
