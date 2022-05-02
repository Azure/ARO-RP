package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func main() {
	b64, _ := ioutil.ReadFile("./test.b64")
	client := &http.Client{
		Timeout: time.Second * 3,
	}
	validate(client, string(b64), map[string]bool{"cloud.openshift.com": true})
}

type requestDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ociRegClient struct {
	httpClient  requestDoer
	serviceURL  string
	service     string
	bearerRealm string
	ctx         context.Context
}

func decodeResponse(body io.Reader) (string, error) {
	response := struct {
		Token string `json:"token"`
	}{}
	err := json.NewDecoder(body).Decode(&response)
	if err != nil {
		return "", err
	}
	return response.Token, err
}

func validate(client requestDoer, b64 string, ignored map[string]bool) {
	log.Println("decoding the base64 credentials")
	jsonBytes, _ := base64.StdEncoding.DecodeString(b64)

	log.Println("decoding the json auth file")
	authsStruct := jsonToAuthStruct(jsonBytes)
	errs := authsStruct.validateCredentials(client, ignored)
	if len(errs) != 0 {
		log.Println("some credentials were not valid")
		for _, v := range errs {
			log.Println(v)
		}
		return
	}
	log.Println("success")
}

func (client *ociRegClient) fetchAuthURL() error {
	//ping the service to get the auth url back
	log.Printf("pinging https://%s/v2 to retrieve authentication endpoint", client.serviceURL)
	req, _ := http.NewRequestWithContext(client.ctx, http.MethodGet, "https://"+client.serviceURL+"/v2", nil)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected status code for https://%s/v2 , got %d but expected %d or %d",
			client.serviceURL, resp.StatusCode, http.StatusOK, http.StatusUnauthorized)
	}

	client.bearerRealm, client.service, err = extractValuesFromAuthHeader(resp.Header.Get("WWW-Authenticate"))

	log.Printf("authentication URL returned by %s is %s", client.serviceURL, client.bearerRealm)
	return err
}

func extractValuesFromAuthHeader(header string) (string, string, error) {
	if !strings.HasPrefix(header, "Bearer realm=\"") {
		return "", "", fmt.Errorf("header is missing data")
	}
	bearerRealm := strings.TrimPrefix(header, "Bearer realm=\"")
	bearerRealm = bearerRealm[:strings.Index(bearerRealm, "\"")]
	bearerRealm = strings.TrimSuffix(bearerRealm, "\"")

	splitted := strings.Split(header, ",")
	if len(splitted) < 2 || len(splitted[1]) == 0 {
		return "", "", fmt.Errorf("header is missing data")
	}

	service := strings.Split(header, ",")[1]
	service = strings.TrimSuffix(service, "\"")
	service = strings.TrimPrefix(service, "service=\"")

	if len(service) == 0 || len(bearerRealm) == 0 {
		return bearerRealm, service, fmt.Errorf("header is missing data")
	}

	return bearerRealm, service, nil
}

func (client ociRegClient) getToken(user, password string) (string, error) {
	//	req, _ := http.NewRequest(http.MethodGet, "https://quay.io/v2/auth?account=rh_ee_jfacchet&scope=repository%3Arh_ee_jfacchet%2Fmyfirstrepo%3Apull&service=quay.io", nil)
	req, _ := http.NewRequestWithContext(client.ctx, http.MethodGet, client.bearerRealm, nil)
	query := req.URL.Query()

	query.Set("account", url.QueryEscape(user))
	//query.Set("scope", url.QueryEscape("repository:"+user+"/"+repo+":pull"))
	query.Set("service", client.service)
	req.URL.RawQuery = query.Encode()
	req.SetBasicAuth(user, password)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("unsuscessfull call to %s", client.bearerRealm)
		return "", fmt.Errorf("authentication unsucessful, got status code %d but wanted %d", resp.StatusCode, http.StatusOK)
	}
	log.Printf("suscessfull call to %s", client.bearerRealm)
	return decodeResponse(resp.Body)
}

func (client ociRegClient) tokenIsValid(token, user, repo string) (bool, error) {
	url := fmt.Sprintf("https://%s/v2/%s/%s/manifests/latest", client.serviceURL, user, repo)
	req, err := http.NewRequestWithContext(client.ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := client.httpClient.Do(req)
	if err != nil || response == nil {
		return false, err
	}
	return response.StatusCode == 200, nil
}

type authsStruct struct {
	Auths map[string]Auth `json:"auths"`
}

type Auth struct {
	Auth string `json:"auth"`
}

func userPasswordFromB64(encoded string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}
	index := strings.Index(string(decoded), ":")
	if index <= 0 || index == len(decoded)-1 {
		return "", "", fmt.Errorf("password string is not valid")
	}
	return string(decoded)[:index], string(decoded)[index+1:], nil
}

func (a authsStruct) userPasswordForService(service string) (string, string, error) {
	auth := a.Auths[service]
	return userPasswordFromB64(auth.Auth)
}

func jsonToAuthStruct(jsonBytes []byte) authsStruct {
	authStruct := authsStruct{}
	err := json.Unmarshal(jsonBytes, &authStruct)
	if err != nil {
		log.Println(err)
	}
	return authStruct
}

func (authsStruct *authsStruct) validateCredentials(client requestDoer, ignored map[string]bool) []error {
	expectedResults := 0

	results := make(chan error)
	defer close(results)

	errors := make([]error, 0)
	for k := range authsStruct.Auths {
		// some urls may need to be ignored.
		if ignored[k] {
			continue
		}
		expectedResults++
		go authsStruct.validateCredential(client, k, results)
		//		fmt.Println(k, v)
	}
	for i := 0; i < expectedResults; i++ {
		routineResult := <-results
		if routineResult != nil {
			errors = append(errors, routineResult)
		}
	}
	return errors
}

func (authsStruct *authsStruct) validateCredential(httpclient requestDoer, service string, results chan error) {
	user, password, err := authsStruct.userPasswordForService(service)
	if err != nil {
		results <- err
		return
	}
	client := ociRegClient{
		httpClient: httpclient,
		serviceURL: service,
		ctx:        context.Background(),
	}

	err = client.fetchAuthURL()
	if err != nil {
		results <- err
		return
	}

	_, err = client.getToken(user, password)
	if err != nil {
		results <- err
		return
	}
	results <- nil
}
