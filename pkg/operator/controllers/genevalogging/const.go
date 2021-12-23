package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	kubeNamespace          = "openshift-azure-logging"
	kubeServiceAccount     = "system:serviceaccount:" + kubeNamespace + ":geneva"
	certificatesSecretName = "certificates"

	GenevaCertName = "gcscert.pem"
	GenevaKeyName  = "gcskey.pem"

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

	fluentConf = `
[SERVICE]
	Parsers_File /etc/td-agent-bit/parsers.conf

[INPUT]
	Name systemd
	Tag journald
	DB /var/lib/fluent/journald

[INPUT]
	Name tail
	Tag containers
	Path /var/log/containers/*
	Path_Key path
	DB /var/lib/fluent/containers
	Parser crio

[INPUT]
	Name tail
	Tag audit
	Path /var/log/kube-apiserver/audit.log
	Path_Key path
	DB /var/lib/fluent/audit
	Parser audit

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP
	Remove SYSLOG_FACILITY

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

[FILTER]
	Name nest
	Match audit
	Operation lift
	Nested_under user
	Add_prefix user_

[FILTER]
	Name nest
	Match audit
	Operation lift
	Nested_under impersonatedUser
	Add_prefix impersonatedUser_

[FILTER]
	Name nest
	Match audit
	Operation lift
	Nested_under responseStatus
	Add_prefix responseStatus_

[FILTER]
	Name nest
	Match audit
	Operation lift
	Nested_under objectRef
	Add_prefix objectRef_

[OUTPUT]
	Name forward
	Match *
	Port 24224
`
)
