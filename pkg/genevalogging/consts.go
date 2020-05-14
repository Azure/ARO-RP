package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	KubeNamespace      = "openshift-azure-logging"
	kubeServiceAccount = "system:serviceaccount:" + KubeNamespace + ":geneva"

	fluentbitImageFormat = "%s.azurecr.io/fluentbit:1.3.9-1"
	mdsdImageFormat      = "%s.azurecr.io/genevamdsd:master_285"

	parsersConf = `
[PARSER]
	Name audit
	Format json
	Time_Key stageTimestamp
	Time_Format %Y-%m-%dT%H:%M:%S.%L

[PARSER]
	Name containerpath
	Format regex
	Regex ^/var/log/containers/(?<POD>[^_]+)_(?<NAMESPACE>[^_]+)_(?<CONTAINER>.+)-(?<CONTAINER_ID>[0-9a-f]{64})\.log$

[PARSER]
	Name crio
	Format regex
	Regex ^(?<TIMESTAMP>[^ ]+) [^ ]+ [^ ]+ (?<MESSAGE>.*)$
	Time_Key TIMESTAMP
	Time_Format %Y-%m-%dT%H:%M:%S.%L
`

	journalConf = `
[INPUT]
	Name systemd
	Tag journald
	DB /var/lib/fluent/journald

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP
	Remove SYSLOG_FACILITY

[OUTPUT]
	Name forward
	Port 24224
`

	containersConf = `
[SERVICE]
	Parsers_File /etc/td-agent-bit/parsers.conf

[INPUT]
	Name tail
	Path /var/log/containers/*
	Path_Key path
	Tag containers
	DB /var/lib/fluent/containers
	Parser crio

[FILTER]
	Name parser
	Match containers
	Key_Name path
	Parser containerpath
	Reserve_Data true

[FILTER]
	Name grep
	Match containers
	Regex NAMESPACE ^(?:default|kube-.*|openshift|openshift-.*)$

[OUTPUT]
	Name forward
	Port 24224
`

	auditConf = `
[SERVICE]
	Parsers_File /etc/td-agent-bit/parsers.conf

[INPUT]
	Name tail
	Path /var/log/kube-apiserver/audit*
	Path_Key path
	Tag audit
	DB /var/lib/fluent/audit
	Parser audit

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under user
	Add_prefix user_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under impersonatedUser
	Add_prefix impersonatedUser_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under responseStatus
	Add_prefix responseStatus_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under objectRef
	Add_prefix objectRef_

[OUTPUT]
	Name forward
	Port 24224
`
)
