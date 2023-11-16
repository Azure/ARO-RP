package poc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type miseRequestData struct {
	MiseURL        string
	OriginalURI    string
	OriginalMethod string
	Token          string
}

const (
	miseURL   = "http://localhost:5000/ValidateRequest"
	originURI = "https://server/endpoint"
)

<<<<<<< HEAD
func authenticateWithMISE(ctx context.Context, token, requestMethod string) (int, string, error) {

	requestData := miseRequestData{
=======
func authenticateWithMISE(ctx context.Context, requestMethod string, token string) error {

	requestData := MiseRequestData{
>>>>>>> 6b3ff8572 (add method back)
		MiseURL:        miseURL,
		OriginalURI:    originURI,
		OriginalMethod: requestMethod,
		Token:          token,
	}

	req, err := createMiseHTTPRequest(ctx, requestData)
	if err != nil {
		return 0, "", err
	}

	// TODO(jonachang): need to cache the client when in production.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("error reading response body: %w", err)
	}

	return resp.StatusCode, string(bodyBytes), nil
}

func createMiseHTTPRequest(ctx context.Context, data miseRequestData) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, data.MiseURL, bytes.NewBuffer(nil))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Original-URI", data.OriginalURI)
	req.Header.Set("Original-Method", data.OriginalMethod)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.Token))
	return req, nil
}
