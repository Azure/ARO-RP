package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	SessionName = "session"
	// Expiration time in unix format
	SessionKeyExpires  = "expires"
	sessionKeyState    = "state"
	SessionKeyUsername = "user_name"
	SessionKeyGroups   = "groups"
)

// AAD is responsible for ensuring that we have a valid login session with AAD.
type AAD interface {
	AAD(http.Handler) http.Handler
	CheckAuthentication(http.Handler) http.Handler
	Login(http.ResponseWriter, *http.Request)
	Logout(string) http.Handler
}

type oauther interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

type claims struct {
	Groups            []string `json:"groups,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
}

type aad struct {
	log *logrus.Entry
	env env.Core
	now func() time.Time
	rt  http.RoundTripper

	tenantID    string
	clientID    string
	clientKey   *rsa.PrivateKey
	clientCerts []*x509.Certificate

	store     *sessions.CookieStore
	oauther   oauther
	verifier  oidc.Verifier
	allGroups []string

	sessionTimeout time.Duration
}

func NewAAD(log *logrus.Entry,
	audit *logrus.Entry,
	env env.Core,
	baseAccessLog *logrus.Entry,
	hostname string,
	sessionKey []byte,
	clientID string,
	clientKey *rsa.PrivateKey,
	clientCerts []*x509.Certificate,
	allGroups []string,
	unauthenticatedRouter *mux.Router,
	verifier oidc.Verifier) (*aad, error) {
	if len(sessionKey) != 32 {
		return nil, errors.New("invalid sessionKey")
	}

	endpoint := oauth2.Endpoint{
		AuthURL:  env.Environment().ActiveDirectoryEndpoint + env.TenantID() + "/oauth2/v2.0/authorize",
		TokenURL: env.Environment().ActiveDirectoryEndpoint + env.TenantID() + "/oauth2/v2.0/token",
	}

	a := &aad{
		log: log,
		env: env,
		now: time.Now,
		rt:  http.DefaultTransport,

		tenantID:    env.TenantID(),
		clientID:    clientID,
		clientKey:   clientKey,
		clientCerts: clientCerts,
		store:       sessions.NewCookieStore(sessionKey),
		oauther: &oauth2.Config{
			ClientID:    clientID,
			Endpoint:    endpoint,
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

	unauthenticatedRouter.NewRoute().Methods(http.MethodGet).Path("/callback").Handler(Log(env, audit, baseAccessLog)(http.HandlerFunc(a.callback)))
	unauthenticatedRouter.NewRoute().Methods(http.MethodGet).Path("/api/login").Handler(Log(env, audit, baseAccessLog)(http.HandlerFunc(a.Login)))
	unauthenticatedRouter.NewRoute().Methods(http.MethodPost).Path("/api/logout").Handler(Log(env, audit, baseAccessLog)(a.Logout("/")))

	return a, nil
}

// AAD is the early stage handler which adds a username to the context if it
// can.  It lets the request through regardless (this is so that failures can be
// logged).
func (a *aad) AAD(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := a.store.Get(r, SessionName)
		if err != nil {
			cookieError, ok := err.(securecookie.Error)
			if ok && cookieError != nil && cookieError.IsDecode() {
				cookie := &http.Cookie{
					Name:    SessionName,
					Path:    "/",
					Expires: time.Unix(0, 0),
				}
				http.SetCookie(w, cookie)
				http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
			} else {
				a.internalServerError(w, err)
			}
			return
		}

		expires, ok := session.Values[SessionKeyExpires].(int64)
		if !ok || time.Unix(expires, 0).Before(a.now()) {
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

// CheckAuthentication is the handler which prevents access to requests without
// valid authentication.
func (a *aad) CheckAuthentication(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if ctx.Value(ContextKeyUsername) == nil {
			if r.URL != nil {
				http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Login will redirect the user to a login page.
func (a *aad) Login(w http.ResponseWriter, r *http.Request) {
	a.redirect(w, r)
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

	state := uuid.DefaultGenerator.Generate()

	session.Values = map[interface{}]interface{}{
		sessionKeyState: state,
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

	session.Values[SessionKeyUsername] = claims.PreferredUsername
	session.Values[SessionKeyGroups] = groupsIntersect
	session.Values[SessionKeyExpires] = a.now().Add(a.sessionTimeout).Unix()

	err = session.Save(r, w)
	if err != nil {
		a.internalServerError(w, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
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

	req.Body = io.NopCloser(strings.NewReader(form))
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
