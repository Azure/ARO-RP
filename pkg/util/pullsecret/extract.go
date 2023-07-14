package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type UserPass struct {
	Username string
	Password string
}

func userPassFromBase64(secret string) (*UserPass, error) {
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, errors.New("malformed auth token")
	}

	split := strings.Split(string(decoded), ":")
	if len(split) != 2 {
		return nil, errors.New("auth token not in format of username:password")
	}

	return &UserPass{
		Username: split[0],
		Password: split[1],
	}, nil
}

// Extract decodes a username and password for a given domain from a
// JSON-encoded pull secret (e.g. from docker auth)
func Extract(rawPullSecret, domain string) (*UserPass, error) {
	pullSecrets := &pullSecret{}
	err := json.Unmarshal([]byte(rawPullSecret), pullSecrets)
	if err != nil {
		return nil, errors.New("malformed pullsecret (invalid JSON)")
	}

	auth, ok := pullSecrets.Auths[domain]
	if !ok {
		return nil, fmt.Errorf("missing '%s' key in pullsecret", domain)
	}

	token, ok := auth["auth"]
	if !ok {
		return nil, errors.New("malformed pullsecret (no auth key)")
	}

	return userPassFromBase64(token.(string))
}
