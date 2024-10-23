package token

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/golang-jwt/jwt/v4"

type custom struct {
	ObjectId   string                 `json:"oid"`
	ClaimNames map[string]interface{} `json:"_claim_names"`
	Groups     []string               `json:"groups"`
	jwt.RegisteredClaims
}

// ExtractClaims extracts the "oid", "_claim_names", and "groups" claims from a given access jwtToken and return them as a custom struct
func ExtractClaims(jwtToken string) (*custom, error) {
	p := jwt.NewParser(jwt.WithoutClaimsValidation())
	c := &custom{}
	_, _, err := p.ParseUnverified(jwtToken, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
