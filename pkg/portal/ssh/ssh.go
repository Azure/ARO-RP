package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	sshNewTimeout = time.Minute
)

type SSH struct {
	env           env.Core
	log           *logrus.Entry
	baseAccessLog *logrus.Entry
	l             net.Listener

	elevatedGroupIDs []string

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
	elevatedGroupIDs []string,
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

		elevatedGroupIDs: elevatedGroupIDs,

		dbOpenShiftClusters: dbOpenShiftClusters,
		dbPortal:            dbPortal,

		dialer: dialer,

		baseServerConfig: &cryptossh.ServerConfig{},

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

// New creates a new temporary password from the request params and sends it
// through the writer
func (s *SSH) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 9 {
		http.Error(w, "invalid resourceId", http.StatusBadRequest)
		return
	}

	resourceID := strings.Join(parts[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		http.Error(w, fmt.Sprintf("invalid resourceId %q", resourceID), http.StatusBadRequest)
		return
	}

	mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediatype != "application/json" {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var req *request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Master < 0 || req.Master > 2 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	elevated := len(stringutils.GroupsIntersect(s.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0
	if !elevated {
		s.sendResponse(w, "", "", "", "Elevated access is required.", s.env.IsLocalDevelopmentMode())
		return
	}

	username := r.Context().Value(middleware.ContextKeyUsername).(string)
	username = strings.SplitN(username, "@", 2)[0]

	password := s.dbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  password,
		TTL: int(sshNewTimeout / time.Second),
		Portal: &api.Portal{
			Username: ctx.Value(middleware.ContextKeyUsername).(string),
			ID:       resourceID,
			SSH: &api.SSH{
				Master: req.Master,
			},
		},
	}

	_, err = s.dbPortal.Create(ctx, portalDoc)
	if err != nil {
		s.internalServerError(w, err)
		return
	}

	host := r.Host
	if strings.ContainsRune(r.Host, ':') {
		host, _, err = net.SplitHostPort(r.Host)
		if err != nil {
			s.internalServerError(w, err)
			return
		}
	}

	s.sendResponse(w, host, username, password, "", s.env.IsLocalDevelopmentMode())
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
		"ssh -o UserKnownHostsFile={{.Hostname}}_known_host{{if .IsLocalDevelopmentMode}} -p 2222{{end}} {{.User}}@{{.Hostname}}"
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
	}
	var buff bytes.Buffer
	err = tmp.Execute(&buff, fields{user, host, line, isLocalDevelopmentMode})
	return buff.String(), err
}
