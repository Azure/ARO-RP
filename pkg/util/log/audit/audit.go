package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"sync"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// see pkg/deploy/generator/resources.go#L901
	CloudRoleRP = "rp"

	DefaultLogMessage = "audit event"

	MetadataCreatedTime    = "createdTime"
	MetadataPayload        = "payload"
	MetadataLogKind        = "logKind"
	MetadataAdminOperation = "adminOp"
	MetadataSource         = "source"

	SourceAdminPortal = "aro-admin"
	SourceRP          = "aro-rp"

	EnvKeyAppID               = "envAppID"
	EnvKeyAppVer              = "envAppVer"
	EnvKeyCloudDeploymentUnit = "envCloudDeploymentUnit"
	EnvKeyCloudRole           = "envCloudRole"
	EnvKeyCloudRoleVer        = "envCloudRoleVer"
	EnvKeyCorrelationID       = "envCorrelationID"
	EnvKeyEnvironment         = "envEnvironmentName"
	EnvKeyHostname            = "envHostname"
	EnvKeyIKey                = "envIKey"
	EnvKeyLocation            = "envLocation"

	PayloadKeyCallerIdentities = "payloadCallerIdentities"
	PayloadKeyCategory         = "payloadCategory"
	PayloadKeyNCloud           = "payloadNCloud"
	PayloadKeyOperationName    = "payloadOperationName"
	PayloadKeyResult           = "payloadResult"
	PayloadKeyRequestID        = "payloadRequestID"
	PayloadKeyTargetResources  = "payloadTargetResources"

	IFXAuditCloudVer = 1.0
	IFXAuditName     = "#Ifx.AuditSchema"
	IFXAuditVersion  = 2.1
	IFXAuditLogKind  = "ifxaudit"

	// ifxAuditFlags is a collection of values bit-packed into a 64-bit integer.
	// These properties describe how the event should be processed by the pipeline
	// in an implementation-independent way.
	ifxAuditFlags = 257
)

var (
	// epoch is an unique identifier associated with the current session of the
	// telemetry library running on the platform. It must be stable during a
	// session, and has no implied ordering across sessions.
	epoch = getNewUuid4String()

	// seqNum is used to track absolute order of uploaded events, per session.
	// It is reset when the ARO component is restarted. The first log will have
	// its sequence number set to 1.
	seqNum      uint64
	seqNumMutex sync.Mutex
)

// Wrapper function to github.com/gofrs/uuid to return just a string representation UUID4
func getNewUuid4String() string {
	newUuid4, _ := uuid.NewV4()
	return newUuid4.String()
}

// AddHook modifies logger by adding the payload hook to its list of hooks.
func AddHook(logger *logrus.Logger) {
	logger.AddHook(&payloadHook{
		payload: &Payload{},
	})
}

// payloadHook, when fires, hydrates an IFxAudit log payload using data in a log
// entry.
type payloadHook struct {
	payload *Payload
}

func (payloadHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *payloadHook) Fire(entry *logrus.Entry) error {
	h.payload = &Payload{}

	// Part-A
	h.payload.EnvVer = IFXAuditVersion
	h.payload.EnvName = IFXAuditName

	if v, ok := entry.Data[MetadataCreatedTime].(string); ok {
		h.payload.EnvTime = v
	}

	h.payload.EnvEpoch = epoch
	h.payload.EnvSeqNum = nextSeqNum()

	if v, ok := entry.Data[EnvKeyIKey].(string); ok {
		h.payload.EnvIKey = v
		delete(entry.Data, EnvKeyIKey)
	}

	h.payload.EnvFlags = ifxAuditFlags

	if v, ok := entry.Data[EnvKeyAppID].(string); ok {
		h.payload.EnvAppID = v
		delete(entry.Data, EnvKeyAppID)
	}

	if v, ok := entry.Data[EnvKeyAppVer].(string); ok {
		h.payload.EnvAppVer = v
		delete(entry.Data, EnvKeyAppVer)
	}

	if v, ok := entry.Data[EnvKeyCorrelationID].(string); ok {
		h.payload.EnvCV = v
		delete(entry.Data, EnvKeyCorrelationID)
	}

	if v, ok := entry.Data[EnvKeyEnvironment].(string); ok {
		h.payload.EnvCloudName = v
		h.payload.EnvCloudEnvironment = v
		delete(entry.Data, EnvKeyEnvironment)
	}

	if v, ok := entry.Data[EnvKeyCloudRole].(string); ok {
		h.payload.EnvCloudRole = v
		delete(entry.Data, EnvKeyCloudRole)
	}

	if v, ok := entry.Data[EnvKeyCloudRoleVer].(string); ok {
		h.payload.EnvCloudRoleVer = v
		delete(entry.Data, EnvKeyCloudRoleVer)
	}

	if v, ok := entry.Data[EnvKeyHostname].(string); ok {
		h.payload.EnvCloudRoleInstance = v
		delete(entry.Data, EnvKeyHostname)
	}

	if v, ok := entry.Data[EnvKeyLocation].(string); ok {
		h.payload.EnvCloudLocation = v
		delete(entry.Data, EnvKeyLocation)
	}

	if v, ok := entry.Data[EnvKeyCloudDeploymentUnit].(string); ok {
		h.payload.EnvCloudDeploymentUnit = v
		delete(entry.Data, EnvKeyCloudDeploymentUnit)
	}

	h.payload.EnvCloudVer = IFXAuditCloudVer

	// Part-B
	if ids, ok := entry.Data[PayloadKeyCallerIdentities].([]CallerIdentity); ok {
		h.payload.CallerIdentities = append(h.payload.CallerIdentities, ids...)
		delete(entry.Data, PayloadKeyCallerIdentities)
	}

	if v, ok := entry.Data[PayloadKeyCategory].(string); ok {
		h.payload.Category = v
		delete(entry.Data, PayloadKeyCategory)
	}

	if v, ok := entry.Data[PayloadKeyOperationName].(string); ok {
		h.payload.OperationName = v
		delete(entry.Data, PayloadKeyOperationName)
	}

	if v, ok := entry.Data[PayloadKeyResult].(Result); ok {
		h.payload.Result = v
		delete(entry.Data, PayloadKeyResult)
	}

	if v, ok := entry.Data[PayloadKeyRequestID].(string); ok {
		h.payload.RequestID = v
		delete(entry.Data, PayloadKeyRequestID)
	}

	if rs, ok := entry.Data[PayloadKeyTargetResources].([]TargetResource); ok {
		h.payload.TargetResources = append(h.payload.TargetResources, rs...)
		delete(entry.Data, PayloadKeyTargetResources)
	}

	// add the audit payload
	b, err := json.Marshal(h.payload)
	if err != nil {
		return err
	}
	entry.Data[MetadataPayload] = string(b)

	return nil
}

func nextSeqNum() uint64 {
	seqNumMutex.Lock()
	defer seqNumMutex.Unlock()

	seqNum++
	return seqNum
}
