package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type (
	// Client can delegate token validation to the Mise container.
	Client struct {
		httpClient  *http.Client
		miseAddress string
	}

	// Input is the set of input options for Client.
	Input struct {
		// OriginalUri is the Uri of the original request being validated.
		OriginalUri string

		// OriginalMethod is the method of the original request being validated.
		OriginalMethod string

		// OriginalIpAddress is the IP address of original request being validated.
		OriginalIPAddress string

		// AuthorizationHeader is the authorization header of the original request being validated.
		AuthorizationHeader string

		// ReturnAllActorClaims specifies whether to return all claims from the actor token.
		ReturnAllActorClaims bool

		// ReturnAllSubjectClaims specifies whether to return all claims from the subject token.
		ReturnAllSubjectClaims bool

		// ActorClaimsToReturn specifies the specific claims to return from the actor token if present.
		ActorClaimsToReturn []string

		// SubjectClaimsToReturn specifies the specific claims to return from the subject token.
		SubjectClaimsToReturn []string
	}

	// Result is the authentication result.
	Result struct {
		// ActorClaims is the set of claims extracted from the actor token based on the input options.
		ActorClaims map[string][]string

		// SubjectClaims is the set of claims extracted from the subject token based on the input options.
		SubjectClaims map[string][]string

		// ErrorDescription is the description of the error from validating the token.
		ErrorDescription []string

		// WWWAuthenticate is the value of the WWWAuthenticate header when the request is unauthorized.
		WWWAuthenticate []string

		// StatusCode is the status code that the Mise container returns as a result of validating the token.
		StatusCode int
	}
)

// New creates a Client able to delegate token validation.
func New(httpClient *http.Client, miseAddress string) *Client {
	return &Client{
		httpClient:  httpClient,
		miseAddress: miseAddress,
	}
}

// ValidateRequest transforms the Input object to a request to the Mise container and returns
// the Result object.
func (c Client) ValidateRequest(ctx context.Context, input Input) (Result, error) {
	req, reqErr := createRequest(ctx, c.miseAddress, input)
	if reqErr != nil {
		return Result{}, reqErr
	}

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return Result{}, respErr
	}

	defer resp.Body.Close()

	return parseResponseIntoResult(resp)
}

func createRequest(ctx context.Context, miseAddress string, input Input) (*http.Request, error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, miseAddress, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	req.Header.Add("Authorization", input.AuthorizationHeader)
	req.Header.Add("Original-Uri", input.OriginalUri)
	req.Header.Add("Original-Method", input.OriginalMethod)
	req.Header.Add("X-Forwarded-For", input.OriginalIPAddress)

	if input.ReturnAllActorClaims {
		req.Header.Add("Return-All-Actor-Token-Claims", "1")
	} else {
		for _, val := range input.ActorClaimsToReturn {
			headerKey := "Return-Actor-Token-Claim-" + val
			req.Header.Add(headerKey, "1")
		}
	}

	if input.ReturnAllSubjectClaims {
		req.Header.Add("Return-All-Subject-Token-Claims", "1")
	} else {
		for _, val := range input.SubjectClaimsToReturn {
			headerKey := "Return-Subject-Token-Claim-" + val
			req.Header.Add(headerKey, "1")
		}
	}

	return req, nil
}

func parseResponseIntoResult(response *http.Response) (Result, error) {
	res := Result{
		StatusCode:    response.StatusCode,
		SubjectClaims: map[string][]string{},
		ActorClaims:   map[string][]string{},
	}

	subjectTokenClaimPrefix := "Subject-Token-Claim-"
	actorTokenClaimPrefix := "Actor-Token-Claim-"
	encodedSubjectTokenClaimPrefix := "Subject-Token-Encoded-Claim-"
	encodedActorTokenClaimPrefix := "Actor-Token-Encoded-Claim-"

	if response.StatusCode == http.StatusOK {
		for k, v := range response.Header {
			if strings.HasPrefix(k, subjectTokenClaimPrefix) {
				claim := k[len(subjectTokenClaimPrefix):]
				// Header will come in from the container, for example, as Subject-Token-Claim-oid
				// but go on parsing the response header will canonicalize the header to Subject-Token-Claim-Oid
				// the name should always be lower case.
				claim = strings.ToLower(claim)
				res.SubjectClaims[claim] = append(res.SubjectClaims[claim], v...)
			} else if strings.HasPrefix(k, actorTokenClaimPrefix) {
				claim := k[len(actorTokenClaimPrefix):]
				// See comment above above about casing for Subject token claims
				claim = strings.ToLower(claim)
				res.ActorClaims[claim] = append(res.ActorClaims[claim], v...)
			} else if strings.HasPrefix(k, encodedSubjectTokenClaimPrefix) {
				claim := k[len(encodedSubjectTokenClaimPrefix):]
				// See comment above above about casing for Subject token claims
				claim = strings.ToLower(claim)
				decodedValues, err := decodeClaims(v)
				if err != nil {
					return res, err
				}
				res.SubjectClaims[claim] = append(res.SubjectClaims[claim], decodedValues...)
			} else if strings.HasPrefix(k, encodedActorTokenClaimPrefix) {
				claim := k[len(encodedActorTokenClaimPrefix):]
				// See comment above above about casing for Subject token claims
				claim = strings.ToLower(claim)
				decodedValues, err := decodeClaims(v)
				if err != nil {
					return res, err
				}
				res.ActorClaims[claim] = append(res.ActorClaims[claim], decodedValues...)
			}
		}
	} else {
		res.ErrorDescription = response.Header["Error-Description"]

		if response.StatusCode == http.StatusUnauthorized {
			res.WWWAuthenticate = response.Header["Www-Authenticate"]
		}
	}

	return res, nil
}

func decodeClaims(encodedClaims []string) ([]string, error) {
	values := make([]string, len(encodedClaims))
	for i := range encodedClaims {
		enc, err := base64.StdEncoding.DecodeString(encodedClaims[i])
		if err != nil {
			return nil, err
		}
		values[i] = string(enc)
	}

	return values, nil
}
