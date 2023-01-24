package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/form3tech-oss/jwt-go"
	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
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

func TestNewAAD(t *testing.T) {
	_, err := NewAAD(nil, nil, nil, nil, "", nil, "", nil, nil, nil, nil, nil)
	if err.Error() != "invalid sessionKey" {
		t.Error(err)
	}
}

func TestAAD(t *testing.T) {
	for _, tt := range []struct {
		name              string
		request           func(*aad) (*http.Request, error)
		wantStatusCode    int
		wantAuthenticated bool
		wantUsername      string
		wantGroups        []string
	}{
		{
			name: "authenticated",
			request: func(a *aad) (*http.Request, error) {
				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					SessionKeyUsername: "username",
					SessionKeyGroups:   []string{"group1", "group2"},
					SessionKeyExpires:  int64(1),
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
				}, nil
			},
			wantAuthenticated: true,
			wantUsername:      "username",
			wantGroups:        []string{"group1", "group2"},
		},
		{
			name: "expired - not authenticated",
			request: func(a *aad) (*http.Request, error) {
				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					SessionKeyUsername: "username",
					SessionKeyGroups:   []string{"group1", "group2"},
					SessionKeyExpires:  int64(-1),
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
				}, nil
			},
		},
		{
			name: "no cookie - not authenticated",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{}, nil
			},
		},
		{
			name: "invalid cookie",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Cookie": []string{"session=xxx"},
					},
					URL: &url.URL{Path: ""},
				}, nil
			},
			wantStatusCode: http.StatusTemporaryRedirect,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			env.EXPECT().TenantID().AnyTimes().Return("common")

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), "", nil, nil, nil, mux.NewRouter(), nil)
			if err != nil {
				t.Fatal(err)
			}
			a.now = func() time.Time { return time.Unix(0, 0) }

			var username string
			var usernameok bool
			var groups []string
			var groupsok bool
			h := a.AAD(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				username, usernameok = r.Context().Value(ContextKeyUsername).(string)
				groups, groupsok = r.Context().Value(ContextKeyGroups).([]string)
			}))

			r, err := tt.request(a)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if tt.wantStatusCode != 0 && w.Code != tt.wantStatusCode {
				t.Error(w.Code)
			}

			if username != tt.wantUsername {
				t.Error(username)
			}
			if usernameok != tt.wantAuthenticated {
				t.Error(usernameok)
			}
			if !reflect.DeepEqual(groups, tt.wantGroups) {
				t.Error(groups)
			}
			if groupsok != tt.wantAuthenticated {
				t.Error(groupsok)
			}
		})
	}
}

func TestCheckAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name              string
		request           func(*aad) (*http.Request, error)
		wantStatusCode    int
		wantAuthenticated bool
	}{
		{
			name: "authenticated",
			request: func(a *aad) (*http.Request, error) {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUsername, "user")
				return http.NewRequestWithContext(ctx, http.MethodGet, "/api/info", nil)
			},
			wantAuthenticated: true,
			wantStatusCode:    http.StatusOK,
		},
		{
			name: "not authenticated",
			request: func(a *aad) (*http.Request, error) {
				ctx := context.Background()
				return http.NewRequestWithContext(ctx, http.MethodGet, "/api/info", nil)
			},
			wantStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name: "not authenticated",
			request: func(a *aad) (*http.Request, error) {
				ctx := context.Background()
				return http.NewRequestWithContext(ctx, http.MethodGet, "/callback", nil)
			},
			wantStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name: "invalid cookie",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Cookie": []string{"session=xxx"},
					},
				}, nil
			},
			wantStatusCode: http.StatusForbidden,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			env.EXPECT().TenantID().AnyTimes().Return("common")

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), "", nil, nil, nil, mux.NewRouter(), nil)
			if err != nil {
				t.Fatal(err)
			}

			var authenticated bool
			h := a.CheckAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, authenticated = r.Context().Value(ContextKeyUsername).(string)
			}))

			r, err := tt.request(a)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatusCode {
				t.Error(w.Code, tt.wantStatusCode)
			}

			if authenticated != tt.wantAuthenticated {
				t.Fatal(authenticated)
			}

			if tt.wantStatusCode == http.StatusInternalServerError {
				return
			}
		})
	}
}

func TestLogin(t *testing.T) {
	for _, tt := range []struct {
		name           string
		request        func(*aad) (*http.Request, error)
		wantStatusCode int
	}{
		{
			name: "authenticated",
			request: func(a *aad) (*http.Request, error) {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUsername, "user")
				return http.NewRequestWithContext(ctx, http.MethodGet, "/login", nil)
			},
			wantStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name: "not authenticated",
			request: func(a *aad) (*http.Request, error) {
				ctx := context.Background()
				return http.NewRequestWithContext(ctx, http.MethodGet, "/login", nil)
			},
			wantStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name: "invalid cookie",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Cookie": []string{"session=xxx"},
					},
				}, nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			env.EXPECT().TenantID().AnyTimes().Return("common")

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), "", nil, nil, nil, mux.NewRouter(), nil)
			if err != nil {
				t.Fatal(err)
			}

			h := http.HandlerFunc(a.Login)

			r, err := tt.request(a)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatusCode {
				t.Error(w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusInternalServerError {
				return
			}

			if !strings.HasPrefix(w.Header().Get("Location"), "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=&redirect_uri=https%3A%2F%2F%2Fcallback&response_type=code&scope=openid+profile&state=") {
				t.Error(w.Header().Get("Location"))
			}
		})
	}
}

func TestLogout(t *testing.T) {
	for _, tt := range []struct {
		name           string
		request        func(*aad) (*http.Request, error)
		wantStatusCode int
	}{
		{
			name: "authenticated",
			request: func(a *aad) (*http.Request, error) {
				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					SessionKeyUsername: "username",
					SessionKeyGroups:   []string{"group1", "group2"},
					SessionKeyExpires:  int64(0),
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
				}, nil
			},
			wantStatusCode: http.StatusSeeOther,
		},
		{
			name: "no cookie - not authenticated",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					URL: &url.URL{},
				}, nil
			},
			wantStatusCode: http.StatusSeeOther,
		},
		{
			name: "invalid cookie",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Cookie": []string{"session=xxx"},
					},
				}, nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			env.EXPECT().TenantID().AnyTimes().Return("common")

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), "", nil, nil, nil, mux.NewRouter(), nil)
			if err != nil {
				t.Fatal(err)
			}

			h := a.Logout("/bye")

			r, err := tt.request(a)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatusCode {
				t.Error(w.Code)
			}

			if tt.wantStatusCode == http.StatusInternalServerError {
				return
			}

			if w.Header().Get("Location") != "/bye" {
				t.Error(w.Header().Get("Location"))
			}

			var m map[interface{}]interface{}
			cookies := w.Result().Cookies()
			err = securecookie.DecodeMulti(SessionName, cookies[len(cookies)-1].Value, &m, a.store.Codecs...)
			if err != nil {
				t.Fatal(err)
			}

			if len(m) != 0 {
				t.Error(len(m))
			}
		})
	}
}

func TestCallback(t *testing.T) {
	clientID := "00000000-0000-0000-0000-000000000000"
	groups := []string{
		"00000000-0000-0000-0000-000000000001",
	}
	username := "user"

	idToken, err := json.Marshal(claims{
		Groups:            groups,
		PreferredUsername: username,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name              string
		request           func(*aad) (*http.Request, error)
		oauther           oauther
		verifier          oidc.Verifier
		wantAuthenticated bool
		wantError         string
		wantForbidden     bool
	}{
		{
			name: "success",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther: &noopOauther{
				tokenMap: map[string]interface{}{
					"id_token": string(idToken),
				},
			},
			verifier:          &oidc.NoopVerifier{},
			wantAuthenticated: true,
		},
		{
			name: "fail - invalid cookie",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Cookie": []string{"session=xxx"},
					},
				}, nil
			},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - corrupt sessionKeyState",
			request: func(a *aad) (*http.Request, error) {
				return &http.Request{
					URL: &url.URL{},
				}, nil
			},
			oauther: &noopOauther{},
		},
		{
			name: "fail - state mismatch",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{"bad"},
					},
				}, nil
			},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - error returned",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state":             []string{uuid},
						"error":             []string{"bad things happened."},
						"error_description": []string{"really bad things."},
					},
				}, nil
			},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - oauther failed",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther: &noopOauther{
				err: fmt.Errorf("failed"),
			},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - no idtoken",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther:   &noopOauther{},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - verifier error",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther: &noopOauther{
				tokenMap: map[string]interface{}{"id_token": ""},
			},
			verifier: &oidc.NoopVerifier{
				Err: fmt.Errorf("failed"),
			},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - invalid claims",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther: &noopOauther{
				tokenMap: map[string]interface{}{
					"id_token": "",
				},
			},
			verifier:  &oidc.NoopVerifier{},
			wantError: "Internal Server Error\n",
		},
		{
			name: "fail - group mismatch",
			request: func(a *aad) (*http.Request, error) {
				uuid := uuid.DefaultGenerator.Generate()

				cookie, err := securecookie.EncodeMulti(SessionName, map[interface{}]interface{}{
					sessionKeyState: uuid,
				}, a.store.Codecs...)
				if err != nil {
					return nil, err
				}

				return &http.Request{
					URL: &url.URL{},
					Header: http.Header{
						"Cookie": []string{SessionName + "=" + cookie},
					},
					Form: url.Values{
						"state": []string{uuid},
					},
				}, nil
			},
			oauther: &noopOauther{
				tokenMap: map[string]interface{}{
					"id_token": "null",
				},
			},
			verifier:      &oidc.NoopVerifier{},
			wantForbidden: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			env.EXPECT().TenantID().AnyTimes().Return("common")

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), clientID, clientkey, clientcerts, groups, mux.NewRouter(), tt.verifier)
			if err != nil {
				t.Fatal(err)
			}
			a.now = func() time.Time { return time.Unix(0, 0) }
			a.oauther = tt.oauther

			r, err := tt.request(a)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			a.callback(w, r)

			if tt.wantError != "" {
				if w.Code != http.StatusInternalServerError {
					t.Error(w.Code)
				}

				if w.Body.String() != tt.wantError {
					t.Error(w.Body.String())
				}

				return
			}

			type cookie map[interface{}]interface{}
			var m cookie
			cookies := w.Result().Cookies()
			err = securecookie.DecodeMulti(SessionName, cookies[len(cookies)-1].Value, &m, a.store.Codecs...)
			if err != nil {
				t.Fatal(err)
			}

			switch {
			case tt.wantAuthenticated:
				if w.Code != http.StatusTemporaryRedirect {
					t.Error(w.Code)
				}

				if w.Header().Get("Location") != "/" {
					t.Error(w.Header().Get("Location"))
				}

				for _, l := range deep.Equal(m, cookie{
					SessionKeyExpires:  int64(3600),
					SessionKeyGroups:   groups,
					SessionKeyUsername: username,
				}) {
					t.Error(l)
				}

			case tt.wantForbidden:
				if w.Code != http.StatusForbidden {
					t.Error(w.Code)
				}

				if w.Header().Get("Location") != "/" {
					t.Error(w.Header().Get("Location"))
				}

				for _, l := range deep.Equal(m, cookie{}) {
					t.Error(l)
				}
			default:
				if w.Code != http.StatusTemporaryRedirect {
					t.Error(w.Code)
				}

				if w.Header().Get("Location") != "/authcodeurl" {
					t.Error(w.Header().Get("Location"))
				}
				return
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
	a, err := NewAAD(baseLog, audit, env, baseAccessLog, "", make([]byte, 32), clientID, clientkey, clientcerts, nil, mux.NewRouter(), nil)
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
