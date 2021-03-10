package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/coreos/go-oidc"
	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

const (
	SessionName            = "session"
	SessionKeyExpires      = "expires"
	sessionKeyRedirectPath = "redirect_path"
	sessionKeyState        = "state"
	SessionKeyUsername     = "user_name"
	SessionKeyGroups       = "groups"
)

func init() {
	gob.Register(time.Time{})
}

// AAD is responsible for ensuring that we have a valid login session with AAD.
type AAD interface {
	AAD(http.Handler) http.Handler
	Logout(string) http.Handler
	Redirect(http.Handler) http.Handler
}

type oauther interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

type Verifier interface {
	Verify(context.Context, string) (oidctoken, error)
}

type idTokenVerifier struct {
	*oidc.IDTokenVerifier
}

func (v *idTokenVerifier) Verify(ctx context.Context, rawIDToken string) (oidctoken, error) {
	return v.IDTokenVerifier.Verify(ctx, rawIDToken)
}

type oidctoken interface {
	Claims(interface{}) error
}

func NewVerifier(ctx context.Context, env env.Core, clientID string) (Verifier, error) {
	provider, err := oidc.NewProvider(ctx, env.Environment().ActiveDirectoryEndpoint+env.TenantID()+"/v2.0")
	if err != nil {
		return nil, err
	}

	return &idTokenVerifier{
		provider.Verifier(&oidc.Config{
			ClientID: clientID,
		}),
	}, nil
}

type claims struct {
	Groups            []string `json:"groups,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
}

type aad struct {
	isDevelopmentMode bool
	log               *logrus.Entry
	env               env.Core
	now               func() time.Time
	rt                http.RoundTripper

	tenantID    string
	clientID    string
	clientKey   *rsa.PrivateKey
	clientCerts []*x509.Certificate

	store     *sessions.CookieStore
	oauther   oauther
	verifier  Verifier
	allGroups []string

	sessionTimeout time.Duration
}

func NewAAD(log *logrus.Entry,
	env env.Core,
	baseAccessLog *logrus.Entry,
	hostname string,
	sessionKey []byte,
	clientID string,
	clientKey *rsa.PrivateKey,
	clientCerts []*x509.Certificate,
	allGroups []string,
	unauthenticatedRouter *mux.Router,
	verifier Verifier) (AAD, error) {
	if len(sessionKey) != 32 {
		return nil, errors.New("invalid sessionKey")
	}

	a := &aad{
		isDevelopmentMode: env.IsDevelopmentMode(),
		log:               log,
		env:               env,
		now:               time.Now,
		rt:                http.DefaultTransport,

		tenantID:    env.TenantID(),
		clientID:    clientID,
		clientKey:   clientKey,
		clientCerts: clientCerts,
		store:       sessions.NewCookieStore(sessionKey),
		oauther: &oauth2.Config{
			ClientID:    clientID,
			Endpoint:    microsoft.AzureADEndpoint(env.TenantID()),
			RedirectURL: "https://" + hostname + "/callback",
			Scopes: []string{
				"openid",
				"profile",
			},
		},
		verifier:  verifier,
		allGroups: allGroups,

		sessionTimeout: time.Hour,
	}

	a.store.MaxAge(0)
	a.store.Options.Secure = true
	a.store.Options.HttpOnly = true
	a.store.Options.SameSite = http.SameSiteLaxMode

	unauthenticatedRouter.NewRoute().Methods(http.MethodGet).Path("/callback").Handler(Log(baseAccessLog)(http.HandlerFunc(a.callback)))

	return a, nil
}

// AAD is the early stage handler which adds a username to the context if it
// can.  It lets the request through regardless (this is so that failures can be
// logged).
func (a *aad) AAD(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := a.store.Get(r, SessionName)
		if err != nil {
			a.internalServerError(w, err)
			return
		}

		expires, ok := session.Values[SessionKeyExpires].(time.Time)
		if !ok || expires.Before(a.now()) {
			h.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUsername, session.Values[SessionKeyUsername])
		ctx = context.WithValue(ctx, ContextKeyGroups, session.Values[SessionKeyGroups])
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

// Redirect is the late stage (post logging) handler which redirects to AAD if
// there is no valid user.
func (a *aad) Redirect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if ctx.Value(ContextKeyUsername) != nil {
			h.ServeHTTP(w, r)
			return
		}

		a.redirect(w, r)
	})
}

func (a *aad) Logout(url string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := a.store.Get(r, SessionName)
		if err != nil {
			a.internalServerError(w, err)
			return
		}

		session.Values = nil

		err = session.Save(r, w)
		if err != nil {
			a.internalServerError(w, err)
			return
		}

		http.Redirect(w, r, url, http.StatusSeeOther)
	})
}

func (a *aad) redirect(w http.ResponseWriter, r *http.Request) {
	session, err := a.store.Get(r, SessionName)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	path := r.URL.Path
	if path == "/callback" {
		path = "/"
	}

	state := uuid.Must(uuid.NewV4()).String()

	session.Values = map[interface{}]interface{}{
		sessionKeyRedirectPath: path,
		sessionKeyState:        state,
	}

	err = session.Save(r, w)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	http.Redirect(w, r, a.oauther.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (a *aad) callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, err := a.store.Get(r, SessionName)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	state, ok := session.Values[sessionKeyState].(string)
	if !ok {
		a.redirect(w, r)
		return
	}

	delete(session.Values, sessionKeyState)

	err = session.Save(r, w)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	if r.FormValue("state") != state {
		a.internalServerError(w, errors.New("state mismatch"))
		return
	}

	if r.FormValue("error") != "" {
		err := r.FormValue("error")
		if r.FormValue("error_description") != "" {
			err += ": " + r.FormValue("error_description")
		}

		a.internalServerError(w, errors.New(err))
		return
	}

	cliCtx := context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: roundtripper.RoundTripperFunc(a.clientAssertion),
	})

	token, err := a.oauther.Exchange(cliCtx, r.FormValue("code"))
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		a.internalServerError(w, errors.New("id_token not found"))
		return
	}

	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	var claims claims
	err = idToken.Claims(&claims)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	groupsIntersect := GroupsIntersect(a.allGroups, claims.Groups)
	if len(groupsIntersect) == 0 {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}

	redirectPath, ok := session.Values[sessionKeyRedirectPath].(string)
	if !ok {
		a.internalServerError(w, errors.New("redirect_path not found"))
		return
	}

	delete(session.Values, sessionKeyRedirectPath)
	session.Values[SessionKeyUsername] = claims.PreferredUsername
	session.Values[SessionKeyGroups] = groupsIntersect
	session.Values[SessionKeyExpires] = a.now().Add(a.sessionTimeout)

	err = session.Save(r, w)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	http.Redirect(w, r, redirectPath, http.StatusTemporaryRedirect)
}

// clientAssertion adds a JWT client assertion according to
// https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-certificate-credentials
// Treating this as a RoundTripper is more hackery -- this is because the
// underlying oauth2 library is a little unextensible.
func (a *aad) clientAssertion(req *http.Request) (*http.Response, error) {
	oauthConfig, err := adal.NewOAuthConfig(a.env.Environment().ActiveDirectoryEndpoint, a.tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, a.clientID, a.clientCerts[0], a.clientKey, "unused")
	if err != nil {
		return nil, err
	}

	s := &adal.ServicePrincipalCertificateSecret{
		Certificate: a.clientCerts[0],
		PrivateKey:  a.clientKey,
	}

	err = req.ParseForm()
	if err != nil {
		return nil, err
	}

	err = s.SetAuthenticationValues(sp, &req.Form)
	if err != nil {
		return nil, err
	}

	form := req.Form.Encode()

	req.Body = ioutil.NopCloser(strings.NewReader(form))
	req.ContentLength = int64(len(form))

	return a.rt.RoundTrip(req)
}

func (a *aad) internalServerError(w http.ResponseWriter, err error) {
	a.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func GroupsIntersect(as, bs []string) (gs []string) {
	for _, a := range as {
		for _, b := range bs {
			if a == b {
				gs = append(gs, a)
				break
			}
		}
	}

	return gs
}
