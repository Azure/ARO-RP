package cosmosdb

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Authorizer interface {
	Authorize(*http.Request, string, string)
}

type masterKeyAuthorizer struct {
	masterKey []byte
}

func (a *masterKeyAuthorizer) Authorize(req *http.Request, resourceType, resourceLink string) {
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	h := hmac.New(sha256.New, a.masterKey)
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n\n", strings.ToLower(req.Method), resourceType, resourceLink, strings.ToLower(date))

	req.Header.Set("Authorization", url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))))
	req.Header.Set("x-ms-date", date)
}

func NewMasterKeyAuthorizer(masterKey string) (Authorizer, error) {
	b, err := base64.StdEncoding.DecodeString(masterKey)
	if err != nil {
		return nil, err
	}

	return &masterKeyAuthorizer{masterKey: b}, nil
}

type tokenAuthorizer struct {
	token string
}

func (a *tokenAuthorizer) Authorize(req *http.Request, resourceType, resourceLink string) {
	req.Header.Set("Authorization", url.QueryEscape(a.token))
}

func NewTokenAuthorizer(token string) Authorizer {
	return &tokenAuthorizer{token: token}
}
