package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type custom struct {
	ObjectId   string                 `json:"oid"`
	ClaimNames map[string]interface{} `json:"_claim_names"`
	Groups     []string               `json:"groups"`
	jwt.RegisteredClaims
}

func CreateTestToken(oid string) (string, error) {
	// Define the signing key
	signingKey := []byte("test-secret-key")

	// Create the custom claims
	claims := custom{
		ObjectId: oid,
		ClaimNames: map[string]interface{}{
			"example_claim": "example_value",
		},
		Groups: []string{"group1", "group2"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "test-subject",
			Audience:  []string{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "unique-id",
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}

	return tokenString, nil
}
