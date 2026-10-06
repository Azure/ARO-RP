package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"net"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/proxy"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
)

type SSH struct {
	env           env.Core
	log           *logrus.Entry
	baseAccessLog *logrus.Entry
	l             net.Listener

	dbOpenShiftClusters database.OpenShiftClusters
	dbPortal            database.Portal

	dialer proxy.Dialer

	baseServerConfig *cryptossh.ServerConfig

	hostPubKey cryptossh.PublicKey
}

func New(env env.Core,
	log *logrus.Entry,
	baseAccessLog *logrus.Entry,
	l net.Listener,
	hostKey *rsa.PrivateKey,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbPortal database.Portal,
	dialer proxy.Dialer,
) (*SSH, error) {
	hostPubKey, err := cryptossh.NewPublicKey(&hostKey.PublicKey)
	if err != nil {
		return nil, err
	}

	s := &SSH{
		env:           env,
		log:           log,
		baseAccessLog: baseAccessLog,
		l:             l,

		dbOpenShiftClusters: dbOpenShiftClusters,
		dbPortal:            dbPortal,

		dialer: dialer,

		baseServerConfig: &cryptossh.ServerConfig{
			Config: cryptossh.Config{
				Ciphers:      utilssh.Ciphers(),
				KeyExchanges: utilssh.KexAlgorithms(),
				MACs:         utilssh.MACs(),
			},
			PublicKeyAuthAlgorithms: utilssh.PublicKeyAlgorithms(),
		},
		hostPubKey: hostPubKey,
	}

	signer, err := cryptossh.NewSignerFromSigner(hostKey)
	if err != nil {
		return nil, err
	}

	s.baseServerConfig.AddHostKey(signer)

	return s, nil
}
