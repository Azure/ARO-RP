package dataplane

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

var (
	errAPIVersion          = errors.New("the api-version parameter was not in MSI data plane request")
	errInvalidCtxValueType = errors.New("the identity URL context value is not a string")
	errInvalidDomain       = errors.New("the MSI URL was not the expected domain")
	errInvalidIdentityURL  = errors.New("the identity URL derived from the context key `x-ms-identity-url` could not be parsed")
	errNotHTTPS            = errors.New("the scheme of the MSI URL is not https")
)

// injectIdentityURLPolicy injects the msi url to be used when calling the MSI dataplane swagger api client
type injectIdentityURLPolicy struct {
	nextForTest func(req *policy.Request) (*http.Response, error)
	msiHost     string
}

func (t *injectIdentityURLPolicy) Do(req *policy.Request) (*http.Response, error) {
	// The Context has the identity url that we need to append with the apiVersion
	apiVersion := req.Raw().URL.Query().Get(apiVersionParameter)
	if err := validateApiVersion(apiVersion); err != nil {
		return nil, errAPIVersion
	}

	rawIdentityURL, ok := req.Raw().Context().Value(identityURLKey).(string)
	if !ok {
		return nil, errInvalidCtxValueType
	}
	msiURL, err := url.Parse(rawIdentityURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidIdentityURL, err)
	}

	if err := validateIdentityUrl(msiURL, t.msiHost); err != nil {
		return nil, fmt.Errorf("MSI identity URL: %q is invalid: %w", msiURL, err)
	}

	// Append URL with version and set the IdentityURL to the modified value
	appendAPIVersion(msiURL, apiVersion)
	req.Raw().URL = msiURL
	req.Raw().Host = msiURL.Host

	return t.next(req)
}

// allows to fake the response in test
func (t *injectIdentityURLPolicy) next(req *policy.Request) (*http.Response, error) {
	if t.nextForTest == nil {
		return req.Next()
	}
	return t.nextForTest(req)
}

func appendAPIVersion(u *url.URL, version string) {
	q := u.Query()
	q.Set(apiVersionParameter, version)
	u.RawQuery = q.Encode()
}

func validateApiVersion(version string) error {
	if version == "" {
		return errAPIVersion
	}
	return nil
}

func validateIdentityUrl(u *url.URL, msiEndpoint string) error {
	if u.Scheme != https {
		return fmt.Errorf("%w: %q", errNotHTTPS, u)
	}

	// We expect the host to have a format simliar "test.identity.azure.net"
	// Check the suffix of host to be the same as msiEndpoint
	if !strings.HasSuffix(u.Hostname(), msiEndpoint) {
		return fmt.Errorf("%w: %q", errInvalidDomain, u)
	}

	return nil
}
