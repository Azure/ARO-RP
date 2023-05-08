package ssh

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/proxy"
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

	elevated := len(middleware.GroupsIntersect(s.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0
	if !elevated {
		s.sendResponse(w, &response{
			Error: "Elevated access is required.",
		})
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

	port := ""
	if s.env.IsLocalDevelopmentMode() {
		port = "-p 2222 "
	}

	s.sendResponse(w, &response{
		Command:  fmt.Sprintf("ssh %s%s@%s", port, username, host),
		Password: password,
	})
}

func (s *SSH) sendResponse(w http.ResponseWriter, resp *response) {
	b, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		s.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *SSH) internalServerError(w http.ResponseWriter, err error) {
	s.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
