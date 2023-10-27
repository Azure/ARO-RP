package poc

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type MiseRequestData struct {
	MiseURL        string
	OriginalURI    string
	OriginalMethod string
	Token          string
}

const (
	miseURL   = "http://localhost:5000/ValidateRequest"
	originURI = "https://server/endpoint"
)

func AuthenticateWithMISE(w http.ResponseWriter, r *http.Request) error {
	ctx := context.Background()
	token := extractToken(r.Header)

	requestData := MiseRequestData{
		MiseURL:        miseURL,
		OriginalURI:    originURI,
		OriginalMethod: r.Method,
		Token:          token,
	}

	req, err := createMiseHTTPRequest(ctx, requestData)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	log.Default().Println("Response status: ", resp.Status)

	w.WriteHeader(resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("Unauthorized")
	}
}

func extractToken(h http.Header) string {
	auth := h.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	return strings.TrimSpace(token)
}

func createMiseHTTPRequest(ctx context.Context, data MiseRequestData) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, data.MiseURL, bytes.NewBuffer(nil))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Original-URI", data.OriginalURI)
	req.Header.Set("Original-Method", data.OriginalMethod)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.Token))
	return req, nil
}
