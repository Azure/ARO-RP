package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"

	"github.com/sirupsen/logrus"
)

type admin struct {
	log *logrus.Entry
}

func NewAdmin(log *logrus.Entry) ClientAuthorizer {
	return &admin{
		log: log,
	}
}

func (a *admin) IsAuthorized(cs *tls.ConnectionState) bool {
	a.log.Print("Admin auth is not implemented yet")
	return false
}

func (a *admin) IsReady() bool {
	return true
}
