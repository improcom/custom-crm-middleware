package httputil

import (
	"crm-middleware/db/settings"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecretKey = settings.Key("sys", "jwt-secret")

var (
	errJWTInvalidToken            = errors.New("jwt: invalid token")
	errJWTParseClaims             = errors.New("jwt: failed to parse claims")
	errJWTSecretNotConfigured     = errors.New("jwt: signing secret not configured")
	errJWTUnexpectedSigningMethod = errors.New("jwt: unexpected signing method")
)

// TokenClaims carries the identifiers embedded in a JWT access token.
type TokenClaims struct {
	IntegrationID string `json:"integration_id,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
}

type jwtClaims struct {
	TokenClaims
	jwt.RegisteredClaims
}

func IssueAccessToken(integrationID, clientID string) (string, error) {
	secret := settings.Get[string](jwtSecretKey)
	if len(secret) == 0 {
		return "", errJWTSecretNotConfigured
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		TokenClaims: TokenClaims{
			IntegrationID: integrationID,
			ClientID:      clientID,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}).SignedString([]byte(secret))
}

func ValidateAccessToken(token string) (TokenClaims, error) {
	secret := settings.Get[string](jwtSecretKey)
	if len(secret) == 0 {
		return TokenClaims{}, errJWTSecretNotConfigured
	}

	jwtToken, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errJWTUnexpectedSigningMethod
		}
		return []byte(secret), nil
	})
	if err != nil {
		return TokenClaims{}, err
	}

	if !jwtToken.Valid {
		return TokenClaims{}, errJWTInvalidToken
	}

	c, ok := jwtToken.Claims.(*jwtClaims)
	if !ok {
		return TokenClaims{}, errJWTParseClaims
	}

	return c.TokenClaims, nil
}
