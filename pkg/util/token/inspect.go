package token

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/golang-jwt/jwt/v4"

type custom struct {
	ObjectId string `json:"oid"`
	jwt.StandardClaims
}

// GetObjectId extracts the "oid" claim from a given access jwtToken
func GetObjectId(jwtToken string) (string, error) {
	p := jwt.NewParser(jwt.WithoutClaimsValidation())
	c := &custom{}
	_, _, err := p.ParseUnverified(jwtToken, c)
	if err != nil {
		return "", err
	}
	return c.ObjectId, nil
}
