package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
)

const (
	MetadataCreatedTime = "createdTime"
	MetadataPayload     = "payload"
	MetadataLogKind     = "logKind"
	MetadataSource      = "source"

	SourceAdminPortal = "aro-admin"
	SourceRP          = "aro-rp"

	EnvKeyAppID               = "envAppID"
	EnvKeyAppVer              = "envAppVer"
	EnvKeyCloudDeploymentUnit = "envCloudDeploymentUnit"
	EnvKeyCloudRole           = "envCloudRole"
	EnvKeyCloudRoleVer        = "envCloudRoleVer"
	EnvKeyCorrelationID       = "envCorrelationID"
	EnvKeyIKey                = "envIKey"

	PayloadKeyCallerIdentities = "payloadCallerIdentities"
	PayloadKeyCategory         = "payloadCategory"
	PayloadKeyNCloud           = "payloadNCloud"
	PayloadKeyOperationName    = "payloadOperationName"
	PayloadKeyResult           = "payloadResult"
	PayloadKeyRequestID        = "payloadRequestID"
	PayloadKeyTargetResources  = "payloadTargetResources"

	ifxAuditCloudVer = 1.0
	ifxAuditLogKind  = "ifxaudit"
	ifxAuditName     = "#Ifx.AuditSchema"
	ifxAuditVersion  = 2.1

	// ifxAuditFlags is a collection of values bit-packed into a 64-bit integer.
	// These properties describe how the event should be processed by the pipeline
	// in an implementation-independent way.
	ifxAuditFlags = 257
)

var (
	// epoch is an unique identifier associated with the current session of the
	// telemetry library running on the platform. It must be stable during a
	// session, and has no implied ordering across sessions.
	epoch = uuid.NewV4().String()

	// seqNum is used to track absolute order of uploaded events, per session.
	// It is reset when the ARO component is restarted. The first log will have
	// its sequence number set to 1.
	seqNum      uint64
	seqNumMutex sync.Mutex
)

// NewEntry returns a log entry that embeds the provided logger. It has a hook
// that knows how to hydrate an IFxAudit log payload before logging it.
func NewEntry(env env.Core, logger *logrus.Logger) *logrus.Entry {
	logger.AddHook(&payloadHook{
		payload: &AuditPayload{},
		env:     env,
	})

	return logrus.NewEntry(logger)
}

// payloadHook, when fires, hydrates an IFxAudit log payload using data in a log
// entry.
type payloadHook struct {
	payload *AuditPayload
	env     env.Core
}

func (payloadHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *payloadHook) Fire(entry *logrus.Entry) error {
	payload := *h.payload // shallow copy

	// Part-A
	payload.EnvVer = ifxAuditVersion
	payload.EnvName = ifxAuditName

	logTime := entry.Time.UTC().Format(time.RFC3339)
	payload.EnvTime = logTime

	payload.EnvEpoch = epoch
	payload.EnvSeqNum = nextSeqNum()

	if v, ok := entry.Data[EnvKeyIKey].(string); ok {
		payload.EnvIKey = v
		delete(entry.Data, EnvKeyIKey)
	}

	payload.EnvFlags = ifxAuditFlags

	if v, ok := entry.Data[EnvKeyAppID].(string); ok {
		payload.EnvAppId = v
		delete(entry.Data, EnvKeyAppID)
	}

	if v, ok := entry.Data[EnvKeyAppVer].(string); ok {
		payload.EnvAppVer = v
		delete(entry.Data, EnvKeyAppVer)
	}

	if v, ok := entry.Data[EnvKeyCorrelationID].(string); ok {
		payload.EnvCV = v
		delete(entry.Data, EnvKeyCorrelationID)
	}

	payload.EnvCloudName = h.env.Environment().Name

	if v, ok := entry.Data[EnvKeyCloudRole].(string); ok {
		payload.EnvCloudRole = v
		delete(entry.Data, EnvKeyCloudRole)
	}

	if v, ok := entry.Data[EnvKeyCloudRoleVer].(string); ok {
		payload.EnvCloudRoleVer = v
		delete(entry.Data, EnvKeyCloudRoleVer)
	}

	payload.EnvCloudRoleInstance = h.env.Hostname()
	payload.EnvCloudEnvironment = h.env.Environment().Name
	payload.EnvCloudLocation = h.env.Location()

	if v, ok := entry.Data[EnvKeyCloudDeploymentUnit].(string); ok {
		payload.EnvCloudDeploymentUnit = v
		delete(entry.Data, EnvKeyCloudDeploymentUnit)
	}

	payload.EnvCloudVer = ifxAuditCloudVer

	// Part-B
	if ids, ok := entry.Data[PayloadKeyCallerIdentities].([]*CallerIdentity); ok {
		payload.CallerIdentities = append(payload.CallerIdentities, ids...)
		delete(entry.Data, PayloadKeyCallerIdentities)
	}

	if v, ok := entry.Data[PayloadKeyCategory].(string); ok {
		payload.Category = Category(v)
		delete(entry.Data, PayloadKeyCategory)
	}

	if v, ok := entry.Data[PayloadKeyOperationName].(string); ok {
		payload.OperationName = v
		delete(entry.Data, PayloadKeyOperationName)
	}

	if v, ok := entry.Data[PayloadKeyResult].(*Result); ok {
		payload.Result = v
		delete(entry.Data, PayloadKeyResult)
	}

	if v, ok := entry.Data[PayloadKeyRequestID].(string); ok {
		payload.RequestID = v
		delete(entry.Data, PayloadKeyRequestID)
	}

	if rs, ok := entry.Data[PayloadKeyTargetResources].([]*TargetResource); ok {
		payload.TargetResources = append(payload.TargetResources, rs...)
		delete(entry.Data, PayloadKeyTargetResources)
	}

	// add the audit payload
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	entry.Data[MetadataPayload] = string(b)

	// add non-IFxAudit metadata for our own use
	entry.Data[MetadataCreatedTime] = logTime
	entry.Data[MetadataLogKind] = ifxAuditLogKind

	return nil
}

func nextSeqNum() uint64 {
	seqNumMutex.Lock()
	defer seqNumMutex.Unlock()

	seqNum++
	return seqNum
}
