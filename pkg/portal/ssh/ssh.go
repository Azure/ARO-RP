package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"net"
	"net/http"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/proxy"
	utilssh "github.com/Azure/ARO-RP/pkg/util/ssh"
)

const (
	sshNewTimeout = time.Minute
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

type request struct {
	Master int `json:"master,omitempty"`
}

type response struct {
	Command  string `json:"command,omitempty"`
	Password string `json:"password,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (s *SSH) sendResponse(w http.ResponseWriter, hostname, username, password, error string, isLocalDevelopmentMode bool) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")

	if error != "" {
		err := enc.Encode(response{Error: error})
		if err != nil {
			s.internalServerError(w, err)
		}
		return
	}
	command, err := createLoginCommand(isLocalDevelopmentMode, username, hostname, s.hostPubKey)
	resp := response{Command: command, Password: password}
	if err != nil {
		s.internalServerError(w, err)
	}
	err = enc.Encode(resp)
	if err != nil {
		s.internalServerError(w, err)
	}
}

func (s *SSH) internalServerError(w http.ResponseWriter, err error) {
	s.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

const (
	sshCommand = "echo '{{ .KnownHostLine }}' > {{.Hostname}}_known_host ; " +
		"ssh " +
		"-o UserKnownHostsFile={{.Hostname}}_known_host " +
		"-o Ciphers={{ .Ciphers }} " +
		"-o HostKeyAlgorithms={{ .HostKeyAlgorithms }} " +
		"-o KexAlgorithms={{ .KexAlgorithms }} " +
		"-o MACs={{ .MACs }}" +
		"{{if .IsLocalDevelopmentMode}} -p 2222{{end}} {{.User}}@{{.Hostname}}"
)

func createLoginCommand(isLocalDevelopmentMode bool, user, host string, publicKey cryptossh.PublicKey) (string, error) {
	line := knownhosts.Line([]string{host}, publicKey)
	tmp := template.New("command")
	tmp, err := tmp.Parse(sshCommand)
	if err != nil {
		return "", err
	}
	type fields struct {
		User                   string
		Hostname               string
		KnownHostLine          string
		IsLocalDevelopmentMode bool
		Ciphers                string
		HostKeyAlgorithms      string
		KexAlgorithms          string
		MACs                   string
	}
	var buff bytes.Buffer
	err = tmp.Execute(&buff, fields{
		user,
		host,
		line,
		isLocalDevelopmentMode,
		// Assume we want to use the first supported algorithm
		utilssh.Ciphers()[0],
		utilssh.HostKeyAlgorithms()[0],
		utilssh.KexAlgorithms()[0],
		utilssh.MACs()[0],
	})
	return buff.String(), err
}
