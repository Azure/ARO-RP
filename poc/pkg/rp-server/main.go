package main

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

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	t := extractToken(r.Header)
	m := MiseRequestData{
		MiseURL:        "http://localhost:5000/ValidateRequest",
		OriginalURI:    "https://server/endpoint",
		OriginalMethod: r.Method,
		Token:          t,
	}
	req, err := createMiseHTTPRequest(ctx, m)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	log.Default().Println("Response status: ", resp.Status)

	w.WriteHeader(resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintln(w, "Authorized")
	default:
		fmt.Fprintln(w, "Unauthorized")
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
		log.Fatal(err)
		return nil, err
	}
	req.Header.Set("Original-URI", data.OriginalURI)
	req.Header.Set("Original-Method", data.OriginalMethod)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.Token))
	return req, nil
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
