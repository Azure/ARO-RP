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
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	SessionName = "session"
	OIDCCookie  = "oidc_cookie"
	stateCookie = "oidc_state_cookie"
	// Expiration time in unix format
	SessionKeyExpires  = "expires"
	sessionKeyState    = "state"
	SessionKeyUsername = "user_name"
	SessionKeyGroups   = "groups"
)

// AAD is responsible for ensuring that we have a valid login session with AAD.
type AAD interface {
	AAD(http.Handler) http.Handler
	Callback(w http.ResponseWriter, r *http.Request)
	Login(http.ResponseWriter, *http.Request)
	Logout(string) http.Handler
}

type oauther interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

// this interface's only purpose is testability because generating JWT is hard
type accessValidator interface {
	validateAccess(ctx context.Context, rawIDToken string, now time.Time) (forbidden bool, username string, groups []string, expiry time.Time, err error)
}

type defaultAccessValidator struct {
	allowedGroups []string
	verifier      *oidc.IDTokenVerifier
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

	oauther         oauther
	accessValidator accessValidator
	allGroups       []string

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
	verifier *oidc.IDTokenVerifier,
) (*aad, error) {
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
		oauther: &oauth2.Config{
			ClientID:    clientID,
			Endpoint:    endpoint,
			RedirectURL: "https://" + hostname + "/callback",
			Scopes: []string{
				"openid",
				"profile",
			},
		},
		accessValidator: defaultAccessValidator{allowedGroups: allGroups, verifier: verifier},
		allGroups:       allGroups,

		sessionTimeout: time.Hour,
	}

	return a, nil
}

// AAD is the early stage handler which adds a username to the context if it
// can.  It lets the request through regardless (this is so that failures can be
// logged).
// Did we though? checkauth didn't do any logging and redirected to login
func (a *aad) AAD(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oidcToken, err := r.Cookie(OIDCCookie)
		if err != nil {
			http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
			return
		}
		forbidden, username, groups, _, err := a.accessValidator.validateAccess(r.Context(), oidcToken.Value, time.Now())
		if forbidden {
			a.log.Debug(groups)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if err != nil {
			http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUsername, username)
		ctx = context.WithValue(ctx, ContextKeyGroups, groups)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

// Login will redirect the user to a OAUTH login page with a state (CSRF protection).
func (a *aad) Login(w http.ResponseWriter, r *http.Request) {
	state := uuid.DefaultGenerator.Generate()

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, a.oauther.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (a *aad) Logout(url string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: OIDCCookie, MaxAge: -1})
		http.Redirect(w, r, url, http.StatusSeeOther)
	})
}

func (a *aad) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state, err := r.Cookie(stateCookie)
	if err != nil {
		http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
		return
	}

	if r.FormValue("state") != state.Value {
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

	_, _, _, expiry, err := a.accessValidator.validateAccess(r.Context(), rawIDToken, a.now())
	if err != nil {
		// we could not identify the user so we make them try to login again
		http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
		return
	}

	// only keep the cookie for the validity time of the token
	http.SetCookie(w, &http.Cookie{
		Name:     OIDCCookie,
		Value:    rawIDToken,
		Expires:  expiry,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	})

	// delete the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   stateCookie,
		Value:  "",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func extractInfoFromToken(token *oidc.IDToken) (preferredUsername string, groups []string, err error) {
	var claims claims

	err = token.Claims(&claims)
	if err != nil {
		return "", nil, err
	}

	return claims.PreferredUsername, claims.Groups, nil
}

func (d defaultAccessValidator) validateAccess(ctx context.Context, rawIDToken string, now time.Time) (forbidden bool, username string, groups []string, expiry time.Time, err error) {
	idToken, err := d.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return false, "", nil, time.Time{}, err
	}

	if idToken.Expiry.Before(now) {
		// token has expired
		return false, "", nil, idToken.Expiry, nil
	}
	username, groups, err = extractInfoFromToken(idToken)
	if err != nil {
		return false, "", nil, time.Time{}, err
	}
	groupsIntersect := GroupsIntersect(d.allowedGroups, groups)
	if len(groupsIntersect) == 0 {
		return true, username, groups, idToken.Expiry, nil
	}
	return false, username, groups, idToken.Expiry, nil
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
