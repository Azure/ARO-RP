package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

func unmarshalReview(body io.Reader) (admissionv1.AdmissionReview, error) {
	result := admissionv1.AdmissionReview{}
	decoder := json.NewDecoder(body)

	err := decoder.Decode(&result)

	return result, err
}

func unmarshalRequestToSecret(request *admissionv1.AdmissionRequest) (corev1.Secret, error) {
	secret := corev1.Secret{}

	return secret, json.Unmarshal(request.Object.Raw, &secret)
}

func (client ociRegClient) validateSecret(log *logrus.Entry, review admissionv1.AdmissionReview) error {
	secret, err := unmarshalRequestToSecret(review.Request)
	if err != nil {
		return err
	}

	if secret.Type != "kubernetes.io/dockerconfigjson" && secret.Type != "kubernetes.io/dockercfg" {
		//secret is not a pullsecret, so accepting it
		return nil
	}

	credentials := ""
	if v, ok := secret.Data[".dockerconfigjson"]; ok {
		credentials = string(v)
	} else if v, ok := secret.Data[".dockercfg"]; ok {
		credentials = string(v)
	} else {
		return fmt.Errorf("%s did not have a dockerconfigjson or dockercfg field", review.Request.Name)
	}

	errs := client.validate(log, credentials)
	if len(errs) != 0 {
		return fmt.Errorf("some credentials in the pull secret were not valid")
	}

	return nil
}

type requestDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ociRegClient struct {
	httpClient requestDoer
	ctx        context.Context
	required   map[string]bool
	log        *logrus.Entry
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

func (client ociRegClient) validate(log *logrus.Entry, b64 string) []error {
	authsStruct := jsonToAuthStruct([]byte(b64))
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
		log.Println(err)
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
	//	req, _ := http.NewRequest(http.MethodGet, "https://quay.io/v2/auth?account=rh_ee_jfacchet&scope=repository%3Arh_ee_jfacchet%2Fmyfirstrepo%3Apull&service=quay.io", nil)
	req, _ := http.NewRequestWithContext(client.ctx, http.MethodGet, bearerRealm, nil)
	query := req.URL.Query()

	query.Set("account", url.QueryEscape(user))
	//query.Set("scope", url.QueryEscape("repository:"+user+"/"+repo+":pull"))
	query.Set("service", serviceName)
	req.URL.RawQuery = query.Encode()
	req.SetBasicAuth(user, password)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("unsuscessfull call to %s", bearerRealm)
		return "", fmt.Errorf("authentication unsucessful, got status code %d but wanted %d", resp.StatusCode, http.StatusOK)
	}
	log.Printf("suscessfull call to %s", bearerRealm)
	return decodeResponse(resp.Body)
}

func (client ociRegClient) tokenIsValid(url, token, user, repo string) (bool, error) {
	manifestUrl := fmt.Sprintf("https://%s/v2/%s/%s/manifests/latest", url, user, repo)
	req, err := http.NewRequestWithContext(client.ctx, http.MethodGet, manifestUrl, nil)
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

//userPasswordFromB64 extracts username and password from the
//base64 encoded field
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

func jsonToAuthStruct(jsonBytes []byte) authsStruct {
	authStruct := authsStruct{}
	err := json.Unmarshal(jsonBytes, &authStruct)
	if err != nil {
		log.Println(err)
	}
	return authStruct
}

//validateCredentials validates the required credentials from authStruct which
//in parallel
func (client ociRegClient) validateCredentials(authsStruct *authsStruct) []error {
	expectedResults := 0

	results := make(chan error)
	defer close(results)

	for k := range authsStruct.Auths {
		// some urls may need to be ignored.
		if !client.required[k] {
			continue
		}
		expectedResults++
		go client.validateCredential(client.log, *authsStruct, k, results)
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
func (client ociRegClient) validateCredential(log *logrus.Entry, authsStruct authsStruct, service string, results chan error) {
	user, password, err := userPasswordFromB64(authsStruct.Auths[service].Auth)
	if err != nil {
		results <- err
		return
	}

	bearerRealm, serviceName, err := client.fetchAuthURL(client.log, service)
	if err != nil {
		log.Printf("error while fetching auth url: %s\n", err.Error())
		results <- err
		return
	}

	_, err = client.getToken(client.log, bearerRealm, serviceName, user, password)
	if err != nil {
		log.Printf("error while getting the token: %s", err.Error())
		results <- err
		return
	}
	results <- nil
}
