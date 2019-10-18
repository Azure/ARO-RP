package cosmosdb

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/ugorji/go/codec"
)

// Options represents API options
type Options struct {
	PreTriggers  []string
	PostTriggers []string
}

// Error represents an error
type Error struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%d %s: %s", e.StatusCode, e.Code, e.Message)
}

// IsErrorStatusCode returns true if err is of type Error and its StatusCode
// matches statusCode
func IsErrorStatusCode(err error, statusCode int) bool {
	if err, ok := err.(Error); ok {
		return err.StatusCode == statusCode
	}
	return false
}

// ErrETagRequired is the error returned if the ETag field is not populate on a
// PUT or DELETE operation
var ErrETagRequired = fmt.Errorf("ETag is required")

// RetryOnPreconditionFailed retries a function if it fails due to
// PreconditionFailed
func RetryOnPreconditionFailed(f func() error) (err error) {
	for i := 0; i < 5; i++ {
		err = f()
		if !IsErrorStatusCode(err, http.StatusPreconditionFailed) {
			return
		}
		time.Sleep(time.Duration(100*i) * time.Millisecond)
	}
	return
}

// JSONHandle exposes the encode/decode options used by
// github.com/ugorji/go/codec
var JSONHandle = &codec.JsonHandle{
	BasicHandle: codec.BasicHandle{
		DecodeOptions: codec.DecodeOptions{
			ErrorIfNoField: true,
		},
	},
}

func (c *databaseClient) authorizeRequest(req *http.Request, resourceType, resourceLink string) {
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	h := hmac.New(sha256.New, c.masterKey)
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n\n", strings.ToLower(req.Method), resourceType, resourceLink, strings.ToLower(date))

	req.Header.Set("Authorization", url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))))
	req.Header.Set("x-ms-date", date)
}

func (c *databaseClient) do(method, path, resourceType, resourceLink string, expectedStatusCode int, in, out interface{}, headers http.Header) error {
	req, err := http.NewRequest(method, "https://"+c.databaseAccount+".documents.azure.com/"+path, nil)
	if err != nil {
		return err
	}

	if in != nil {
		buf := &bytes.Buffer{}
		err := codec.NewEncoder(buf, JSONHandle).Encode(in)
		if err != nil {
			return err
		}
		req.Body = ioutil.NopCloser(buf)
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header[textproto.CanonicalMIMEHeaderKey(k)] = v
	}

	req.Header.Set("x-ms-version", "2018-12-31")

	c.authorizeRequest(req, resourceType, resourceLink)

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if headers != nil {
		for k := range headers {
			delete(headers, k)
		}
		for k, v := range resp.Header {
			headers[k] = v
		}
	}

	d := codec.NewDecoder(resp.Body, JSONHandle)

	if resp.StatusCode != expectedStatusCode {
		var err Error
		if resp.Header.Get("Content-Type") == "application/json" {
			d.Decode(&err)
		}
		err.StatusCode = resp.StatusCode
		return err
	}

	if out != nil && resp.Header.Get("Content-Type") == "application/json" {
		return d.Decode(&out)
	}

	return nil
}

func setOptions(options *Options, headers http.Header) {
	if len(options.PreTriggers) > 0 {
		headers.Set("X-Ms-Documentdb-Pre-Trigger-Include", strings.Join(options.PreTriggers, ","))
	}
	if len(options.PostTriggers) > 0 {
		headers.Set("X-Ms-Documentdb-Post-Trigger-Include", strings.Join(options.PostTriggers, ","))
	}
}
