package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
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
	epoch = uuid.DefaultGenerator.Generate()

	// seqNum is used to track absolute order of uploaded events, per session.
	// It is reset when the ARO component is restarted. The first log will have
	// its sequence number set to 1.
	seqNum      uint64
	seqNumMutex sync.Mutex
)

// PayloadHook, when fires, hydrates an IFxAudit log payload using data in a log
// entry.
type PayloadHook struct {
	Payload *Payload
}

func (PayloadHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *PayloadHook) Fire(entry *logrus.Entry) error {
	h.Payload = &Payload{}

	// Part-A
	h.Payload.EnvVer = IFXAuditVersion
	h.Payload.EnvName = IFXAuditName

	if v, ok := entry.Data[MetadataCreatedTime].(string); ok {
		h.Payload.EnvTime = v
	}

	h.Payload.EnvEpoch = epoch
	h.Payload.EnvSeqNum = nextSeqNum()

	if v, ok := entry.Data[EnvKeyIKey]; ok {
		h.Payload.EnvIKey = fmt.Sprint(v)
		delete(entry.Data, EnvKeyIKey)
	}

	h.Payload.EnvFlags = ifxAuditFlags

	if v, ok := entry.Data[EnvKeyAppID]; ok {
		h.Payload.EnvAppID = fmt.Sprint(v)
		delete(entry.Data, EnvKeyAppID)
	}

	if v, ok := entry.Data[EnvKeyAppVer]; ok {
		h.Payload.EnvAppVer = fmt.Sprint(v)
		delete(entry.Data, EnvKeyAppVer)
	}

	if v, ok := entry.Data[EnvKeyCorrelationID]; ok {
		h.Payload.EnvCV = fmt.Sprint(v)
		delete(entry.Data, EnvKeyCorrelationID)
	}

	if v, ok := entry.Data[EnvKeyEnvironment]; ok {
		h.Payload.EnvCloudName = fmt.Sprint(v)
		h.Payload.EnvCloudEnvironment = fmt.Sprint(v)
		delete(entry.Data, EnvKeyEnvironment)
	}

	if v, ok := entry.Data[EnvKeyCloudRole]; ok {
		h.Payload.EnvCloudRole = fmt.Sprint(v)
		delete(entry.Data, EnvKeyCloudRole)
	}

	if v, ok := entry.Data[EnvKeyCloudRoleVer]; ok {
		h.Payload.EnvCloudRoleVer = fmt.Sprint(v)
		delete(entry.Data, EnvKeyCloudRoleVer)
	}

	if v, ok := entry.Data[EnvKeyHostname]; ok {
		h.Payload.EnvCloudRoleInstance = fmt.Sprint(v)
		delete(entry.Data, EnvKeyHostname)
	}

	if v, ok := entry.Data[EnvKeyLocation]; ok {
		h.Payload.EnvCloudLocation = fmt.Sprint(v)
		delete(entry.Data, EnvKeyLocation)
	}

	if v, ok := entry.Data[EnvKeyCloudDeploymentUnit]; ok {
		h.Payload.EnvCloudDeploymentUnit = fmt.Sprint(v)
		delete(entry.Data, EnvKeyCloudDeploymentUnit)
	}

	h.Payload.EnvCloudVer = IFXAuditCloudVer

	// Part-B
	if ids, ok := entry.Data[PayloadKeyCallerIdentities].([]CallerIdentity); ok {
		h.Payload.CallerIdentities = append(h.Payload.CallerIdentities, ids...)
		delete(entry.Data, PayloadKeyCallerIdentities)
	}

	if v, ok := entry.Data[PayloadKeyCategory]; ok {
		h.Payload.Category = fmt.Sprint(v)
		delete(entry.Data, PayloadKeyCategory)
	}

	if v, ok := entry.Data[PayloadKeyOperationName]; ok {
		h.Payload.OperationName = fmt.Sprint(v)
		delete(entry.Data, PayloadKeyOperationName)
	}

	if v, ok := entry.Data[PayloadKeyResult].(Result); ok {
		h.Payload.Result = v
		delete(entry.Data, PayloadKeyResult)
	}

	if v, ok := entry.Data[PayloadKeyRequestID]; ok {
		h.Payload.RequestID = fmt.Sprint(v)
		delete(entry.Data, PayloadKeyRequestID)
	}

	if rs, ok := entry.Data[PayloadKeyTargetResources].([]TargetResource); ok {
		h.Payload.TargetResources = append(h.Payload.TargetResources, rs...)
		delete(entry.Data, PayloadKeyTargetResources)
	}

	// add the audit payload
	b, err := json.Marshal(h.Payload)
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
