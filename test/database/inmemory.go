package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
)

var fakeCode []byte = []byte{'F', 'A', 'K', 'E'}

type fakeCipher struct {
}

func (c fakeCipher) Decrypt(in []byte) ([]byte, error) {
	return in[4:], nil
}

func (c fakeCipher) Encrypt(in []byte) ([]byte, error) {
	out := make([]byte, 4+len(in))
	_ = copy(out, fakeCode)
	_ = copy(out[4:], in)
	return out, nil
}

func NewDatabase(ctx context.Context, log *logrus.Entry) (*database.Database, string, error) {
	cipher := &fakeCipher{}
	uuid := uuid.NewV4().String()
	h := database.NewJSONHandle(cipher)

	osc := newOpenShiftClusters(h)
	sub := newSubscriptions(h)
	bil := newBilling(h)

	db := &database.Database{
		OpenShiftClusters: database.NewOpenShiftClustersWithProvidedClient(uuid, osc, nil),
		Subscriptions:     database.NewSubscriptionsWithProvidedClient(uuid, sub),
		Billing:           database.NewBillingWithProvidedClient(uuid, bil),
	}

	return db, uuid, nil
}
