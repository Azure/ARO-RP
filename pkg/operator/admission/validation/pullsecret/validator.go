package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
)

const (
	ocmKey string = "registry.redhat.io"
	aroKey string = "arosvc.azurecr.io"
)

type RequestDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ociRegClient struct {
	httpClient RequestDoer
	ctx        context.Context
	required   map[string]bool
	log        *logrus.Entry
}

func unmarshalReview(body io.Reader) (admissionv1.AdmissionReview, error) {
	result := admissionv1.AdmissionReview{}
	decoder := json.NewDecoder(body)

	err := decoder.Decode(&result)

	return result, err
}

func (client ociRegClient) validateSecret(log *logrus.Entry, review admissionv1.AdmissionReview) error {
	secret, oldSecret, err := unmarshalRequestToSecret(review.Request)
	if err != nil {
		return err
	}

	name := secret.Name
	if name == "" {
		//in case of deletion, only old is present
		name = oldSecret.Name
	}

	if name != "pull-secret" {
		//secret is not the one we are watching
		return nil
	}

	secretType := secret.Type
	if secretType == "" {
		//in case of deletion, only old is present
		secretType = oldSecret.Type
	}
	if secretType != "kubernetes.io/dockerconfigjson" && secretType != "kubernetes.io/dockercfg" {
		//secret is not a pullsecret, so accepting it
		return nil
	}

	authsStructNew, authsStructOld, err := extractAuthsFromSecrets(secret, oldSecret, review.Request.Operation)
	if err != nil {
		return err
	}

	isOCM, err := basicAuthValidation(authsStructNew, authsStructOld, review.Request.Operation, client.required)
	if err != nil {
		return err
	}
	if !isOCM {
		return nil
	}

	errs := client.validateLogErrors(log, authsStructNew)
	if len(errs) != 0 {
		return errors.New("some credentials in the pullsecret were not valid")
	}

	return nil
}

func decodeResponse(body io.Reader) (string, error) {
	response := struct {
		Token string `json:"token"`
	}{}
	err := json.NewDecoder(body).Decode(&response)
	if err != nil {
		return "", err
	}
	if response.Token == "" {
		return "", fmt.Errorf("no token in response")
	}
	return response.Token, err
}

func (client ociRegClient) validateLogErrors(log *logrus.Entry, authsStruct authsStruct) []error {
	errs := client.validateCredentials(&authsStruct)
	for _, v := range errs {
		log.Println(v)
	}
	return errs
}

func (client ociRegClient) fetchAuthURL(log *logrus.Entry, url string) (string, string, error) {
	//ping the service to get the auth url back
	log.Printf("pinging https://%s/v2 to retrieve authentication endpoint", url)
	req, err := http.NewRequestWithContext(client.ctx, http.MethodGet, "https://"+url+"/v2", nil)
	if err != nil {
		return "", "", err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return "", "",
			fmt.Errorf("unexpected status code for https://%s/v2 , got %d but expected %d or %d",
				url, resp.StatusCode, http.StatusOK, http.StatusUnauthorized)
	}

	bearerRealm, service, err := extractValuesFromAuthHeader(resp.Header.Get("WWW-Authenticate"))

	log.Printf("authentication URL returned by %s is %s", url, bearerRealm)
	return bearerRealm, service, err
}

func extractValuesFromAuthHeader(header string) (string, string, error) {
	//if this is not good enough we could get the code from podman.
	//it is however a lot bigger
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

func (client ociRegClient) getToken(log *logrus.Entry, bearerRealm, serviceName, user, password string) (string, error) {
	req, _ := http.NewRequestWithContext(client.ctx, http.MethodGet, bearerRealm, nil)
	query := req.URL.Query()

	query.Set("account", url.QueryEscape(user))
	query.Set("service", serviceName)
	req.URL.RawQuery = query.Encode()
	req.SetBasicAuth(user, password)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("unsuscessfull call to %s", bearerRealm)
		return "", fmt.Errorf("authentication unsucessful, got status code %d but wanted %d", resp.StatusCode, http.StatusOK)
	}
	log.Printf("suscessfull call to %s", bearerRealm)
	return decodeResponse(resp.Body)
}

//validateCredentials validates the required credentials from authStruct which
//in parallel
func (client ociRegClient) validateCredentials(authsStruct *authsStruct) []error {
	expectedResults := 0

	results := make(chan error)
	defer close(results)

	for k := range authsStruct.Auths {
		// some urls may need to be ignored.
		if client.required[k] {
			expectedResults++
			go client.validateCredential(client.log, *authsStruct, k, results)
		}
	}

	errors := make([]error, 0)
	for i := 0; i < expectedResults; i++ {
		routineResult := <-results
		if routineResult != nil {
			errors = append(errors, routineResult)
		}
	}
	return errors
}

//validateCredential checks if the credentials in authStruct are valid for service.
//if there is any error it writes it to the channel
func (client ociRegClient) validateCredential(log *logrus.Entry, authsStruct authsStruct, service string, results chan<- error) {
	user, password, err := userPasswordFromB64(authsStruct.Auths[service].Auth)
	if err != nil {
		results <- fmt.Errorf("could not read user/password for registry %s. err: %s", service, err)
		return
	}

	bearerRealm, serviceName, err := client.fetchAuthURL(client.log, service)
	if err != nil {
		log.Printf("error while fetching auth url for registry %s: %s\n", service, err)
		results <- err
		return
	}

	_, err = client.getToken(client.log, bearerRealm, serviceName, user, password)
	if err != nil {
		log.Printf("error while getting the token for registry %s: %s", service, err)
		results <- err
		return
	}
	results <- nil
}
