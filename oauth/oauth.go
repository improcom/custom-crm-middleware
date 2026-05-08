package oauth

import (
	"crm-middleware/model"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type ExtraConfig struct {
	Attrs  *attrsCollector
	Expiry time.Duration
}

func (c ExtraConfig) FixedExpiry(expiry time.Time) time.Time {
	if expiry.IsZero() {
		expiry = time.Now().Add(time.Duration(c.Expiry))
	}
	return expiry
}

// Known standard and non-standard OIDC claims for id_token layer that might contain identity.
var IDClaims = []string{"email", "name", "user", "username", "upn", "sub"}

type attrsCollector struct {
	keys []string
}

func AttrsCollector(keys ...string) *attrsCollector {
	return &attrsCollector{
		keys: append(keys, IDClaims...),
	}
}

func (collector *attrsCollector) FromToken(token *oauth2.Token) *model.Attributes {
	var extra = make(map[string][]any)
	for _, key := range collector.keys {
		extra[key] = append(extra[key], token.Extra(key))
	}

	idToken, _ := token.Extra("id_token").(string)
	if parts := strings.Split(idToken, "."); len(parts) > 1 {

		if decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(parts[1]); err == nil {
			var objmap map[string]any

			if err = json.Unmarshal(decoded, &objmap); err == nil {

				for _, key := range collector.keys {
					extra[key] = append(extra[key], objmap[key])
				}
			}
		}
	}

	var attrs = new(model.Attributes)
	for key, values := range extra {
		attrs.Set(key, values...)
	}
	return attrs
}

func (collector *attrsCollector) FromRequest(r *http.Request) *model.Attributes {
	var extra = make(map[string][]any)

	var query = r.URL.Query()
	for _, key := range collector.keys {

		for _, q := range query[key] {
			extra[key] = append(extra[key], q)
		}

		for _, h := range r.Header[key] {
			extra[key] = append(extra[key], h)
		}

		for _, v := range r.Form[key] {
			extra[key] = append(extra[key], v)
		}
	}

	var attrs = new(model.Attributes)
	for key, values := range extra {
		attrs.Set(key, values...)
	}
	return attrs
}

func newToken(token *oauth2.Token, extraConfig *ExtraConfig) *model.Token {
	return &model.Token{
		Type:         token.TokenType,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       extraConfig.FixedExpiry(token.Expiry),

		Attrs: extraConfig.Attrs.FromToken(token),
	}
}

func oauth2Token(t *model.Token) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.Type,
		Expiry:       t.Expiry,
	}
}
