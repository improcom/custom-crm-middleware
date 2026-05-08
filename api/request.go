package api

import (
	"context"
	"crm-middleware/httputil"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func appendIfNotEmpty(attrs []any, args ...slog.Attr) []any {
	for _, a := range args {
		if a.Value.String() != "" {
			attrs = append(attrs, a)
		}
	}
	return attrs
}

type Claims struct {
	ClientID      string `json:"client_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	IntegrationID string `json:"integration_id,omitempty"`
}

func (c Claims) SlogAttr() slog.Attr {
	return slog.Group("claims",
		appendIfNotEmpty([]any{},
			slog.String("integration-id", c.IntegrationID),
			slog.String("client-id", c.ClientID),
			slog.String("user-id", c.UserID),
		)...,
	)
}

type ObjectRef struct {
	ObjectType string
	ObjectID   string
}

func (o ObjectRef) SlogAttr() slog.Attr {
	return slog.Group("object",
		appendIfNotEmpty([]any{},
			slog.String("type", o.ObjectType),
			slog.String("id", o.ObjectID),
		)...,
	)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err := httputil.ValidateAccessToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			httputil.Error(w, "invalid token provided", http.StatusUnauthorized,
				httputil.SlogError(err),
			)
			return
		}

		next.ServeHTTP(w, SetClaims(r, Claims{
			IntegrationID: tok.IntegrationID,
			ClientID:      tok.ClientID,
			UserID:        r.Header.Get("user_id"),
		}))
	})
}

type claimsCtxKey struct{}
type objectCtxKey struct{}

func GetClaims(r *http.Request) Claims {
	v, _ := r.Context().Value(claimsCtxKey{}).(Claims)
	return v
}

func GetObject(r *http.Request) ObjectRef {
	v, _ := r.Context().Value(objectCtxKey{}).(ObjectRef)
	return v
}

func SetClaims(r *http.Request, c Claims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), claimsCtxKey{}, c))
}

func WithObject(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r.WithContext(context.WithValue(r.Context(), objectCtxKey{}, ObjectRef{
			ObjectType: chi.URLParam(r, "object_type"),
			ObjectID:   chi.URLParam(r, "object_id"),
		})))
	}
}
