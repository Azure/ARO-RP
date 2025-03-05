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
		return nil, errors.New("invalid Base64")
	}

	split := strings.SplitN(string(decoded), ":", 2)
	if len(split) != 2 {
		return nil, errors.New("not in format of username:password")
	}

	return &UserPass{
		Username: split[0],
		Password: split[1],
	}, nil
}

// Extract decodes into a map, usernames and corresponding password for
// all domains from a JSON-encoded pull secret (e.g. from docker auth)
func Extract(rawPullSecret string) (map[string]*UserPass, error) {
	pullSecrets := &pullSecret{}
	err := json.Unmarshal([]byte(rawPullSecret), pullSecrets)
	if err != nil {
		return nil, errors.New("malformed pullsecret (invalid JSON)")
	}

	pullSecretMap := make(map[string]*UserPass)
	for key, auth := range pullSecrets.Auths {
		token, ok := auth["auth"]
		if !ok {
			return nil, fmt.Errorf("malformed pullsecret (no auth key) for key %s", key)
		}

		pullSecretMap[key], err = userPassFromBase64(token.(string))
		if err != nil {
			return nil, fmt.Errorf("malformed auth token for key %s: %s", key, err)
		}
	}

	return pullSecretMap, nil
}
