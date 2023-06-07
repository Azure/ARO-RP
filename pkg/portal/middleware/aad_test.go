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
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/golang-jwt/jwt"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var (
	clientkey   *rsa.PrivateKey
	clientcerts []*x509.Certificate
)

func init() {
	var err error

	clientkey, clientcerts, err = utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		panic(err)
	}
}

func TestNewAAD(t *testing.T) {
	_, err := NewAAD(nil, nil, nil, nil, "", nil, "", nil, nil, nil, &oidc.IDTokenVerifier{})
	if err.Error() != "invalid sessionKey" {
		t.Error(err)
	}
}

type fakeAccessValidator struct {
	forbidden bool
	user      string
	expiry    time.Time
	groups    []string
	err       error
}

func (f fakeAccessValidator) validateAccess(ctx context.Context, rawIDToken string, now time.Time) (bool, string, []string, time.Time, error) {
	return f.forbidden, f.user, f.groups, f.expiry, f.err
}

func TestAAD(t *testing.T) {
	var tests = []struct {
		name            string
		userReturn      string
		errReturn       error
		forbiddenReturn bool
		groupsReturn    []string
		wantUser        string
		wantGroups      []string
	}{
		{name: "authenticated", userReturn: "mattHicks", groupsReturn: []string{"ceo"}, wantUser: "mattHicks", wantGroups: []string{"ceo"}},
		{name: "forbidden", forbiddenReturn: true},
		{name: "error", errReturn: errors.New("expired")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aadStruct := &aad{accessValidator: fakeAccessValidator{user: tt.userReturn, groups: tt.groupsReturn, forbidden: tt.forbiddenReturn, err: tt.errReturn}}

			dummyRequest, _ := http.NewRequest(http.MethodGet, "https://redhat.com/hello", nil)
			dummyRequest.AddCookie(&http.Cookie{Name: OIDCCookie, Value: "supertoken"})
			writer := httptest.NewRecorder()

			// this simulates a handler that is called after AAD, to check if we have the right values in the
			// context. It will be called from aadStruct.AAD
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user := r.Context().Value(ContextKeyUsername).(string)
				if user != tt.wantUser {
					t.Errorf("wanted user to be %s but got %s ", tt.wantUser, user)
				}
				groups := r.Context().Value(ContextKeyGroups).([]string)
				if diff := cmp.Diff(groups, tt.wantGroups); diff != "" {
					t.Errorf("unexpected groups value %s", diff)
				}
			})

			aadStruct.AAD(nextHandler).ServeHTTP(writer, dummyRequest)

			if tt.forbiddenReturn && writer.Result().StatusCode != http.StatusForbidden {
				t.Errorf("was expecting status code to be 403 but got %d", writer.Result().StatusCode)
			} else if tt.errReturn != nil && writer.Result().StatusCode != http.StatusTemporaryRedirect {
				t.Errorf("was expecting status code to be 302 but got %d", writer.Result().StatusCode)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://redhat.com", nil)
	if err != nil {
		t.Fatal("could not create the request", err)
	}
	request.AddCookie(&http.Cookie{
		Name:  OIDCCookie,
		Value: "I love potatoes",
	})

	writer := httptest.NewRecorder()

	aadStruct := &aad{}
	aadStruct.Logout("/").ServeHTTP(writer, request)

	found := false
	for _, v := range writer.Result().Cookies() {
		if v.Name == OIDCCookie {
			found = true
			if v.MaxAge != -1 {
				t.Error("cookie does not have expected max age field")
			}
		}
	}

	if !found {
		t.Error("cookie was not found")
	}
}

type noopOauther struct {
	tokenMap map[string]interface{}
	err      error
}

func (noopOauther) AuthCodeURL(string, ...oauth2.AuthCodeOption) string {
	return "authcodeurl"
}

func (o *noopOauther) Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if o.err != nil {
		return nil, o.err
	}

	t := oauth2.Token{}
	return t.WithExtra(o.tokenMap), nil
}

func TestCallback(t *testing.T) {
	var tests = []struct {
		name            string
		wantStatusCode  int
		wantLocation    string
		exchangeError   error
		token           string
		validatorErr    error
		validatorExpiry time.Time
		hasStateCookie  bool
		stateMismatch   bool
		wantOIDCCookie  bool
	}{
		{name: "no state cookie", hasStateCookie: false, wantStatusCode: http.StatusTemporaryRedirect, wantLocation: "/api/login"},
		{name: "state mismatch", stateMismatch: true, hasStateCookie: true, wantStatusCode: http.StatusInternalServerError},
		{name: "exchange error", hasStateCookie: true, wantStatusCode: http.StatusInternalServerError, exchangeError: errors.New("err")},
		{name: "no id token", hasStateCookie: true, wantStatusCode: http.StatusInternalServerError},
		{name: "validator error", hasStateCookie: true, wantStatusCode: http.StatusTemporaryRedirect, wantLocation: "/api/login", token: "supertoken", validatorErr: errors.New("validator error")},
		{name: "all good", hasStateCookie: true, wantStatusCode: http.StatusTemporaryRedirect, wantLocation: "/", token: "supertoken", wantOIDCCookie: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.Out = io.Discard
			aad := &aad{log: logrus.NewEntry(logger), now: time.Now}

			tokenMap := make(map[string]interface{})
			if tt.token != "" {
				tokenMap["id_token"] = tt.token
			}
			aad.oauther = &noopOauther{err: tt.exchangeError, tokenMap: tokenMap}
			aad.accessValidator = fakeAccessValidator{err: tt.validatorErr, expiry: tt.validatorExpiry}

			writer := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/callback", nil)
			if tt.hasStateCookie {
				request.AddCookie(&http.Cookie{Name: stateCookie, Value: "some value"})
			}

			request.Form = map[string][]string{"state": {"some value"}}
			if tt.stateMismatch {
				request.Form = map[string][]string{"state": {"some other value"}}
			}

			aad.Callback(writer, request)

			if writer.Result().StatusCode != tt.wantStatusCode {
				t.Errorf("wanted status code to be %d but got %d", tt.wantStatusCode, writer.Result().StatusCode)
			}
			if tt.wantLocation != writer.Result().Header.Get("Location") {
				t.Errorf("wanted location header to be %s but got %s", tt.wantLocation, writer.Result().Header.Get("Location"))
			}
			if tt.wantOIDCCookie {
				hasCookie := false
				for _, v := range writer.Result().Cookies() {
					if v.Name == OIDCCookie && v.Value == tt.token && v.SameSite == http.SameSiteStrictMode && v.Secure == true && v.HttpOnly == true {
						hasCookie = true
						break
					}
				}
				if !hasCookie {
					t.Error("did not have the right cookie")
				}
			}
		})
	}
}

func TestClientAssertion(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
	env.EXPECT().TenantID().AnyTimes().Return("common")

	clientID := "00000000-0000-0000-0000-000000000000"
	_, audit := testlog.NewAudit()
	_, baseLog := testlog.New()
	_, baseAccessLog := testlog.New()
	a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), clientID, clientkey, clientcerts, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.rt = roundtripper.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	})

	req, err := http.NewRequest(http.MethodGet, "https://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Form = url.Values{"test": []string{"value"}}

	_, err = a.clientAssertion(req)
	if err != nil {
		t.Fatal(err)
	}

	if req.Form.Get("test") != "value" {
		t.Error(req.Form.Get("test"))
	}

	if req.Form.Get("client_assertion_type") != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
		t.Error(req.Form.Get("client_assertion_type"))
	}

	p := &jwt.Parser{}
	_, err = p.Parse(req.Form.Get("client_assertion"), func(*jwt.Token) (interface{}, error) {
		return &clientkey.PublicKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
