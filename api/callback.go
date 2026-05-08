package api

import (
	"crm-middleware/httputil"
	"crm-middleware/model"
	"crm-middleware/oauth"
	"crm-middleware/repo"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
)

func AuthorizationCodeCallbackHandlerFunc(getter func() model.TokenGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			params = r.URL.Query()
			state  = params.Get("state")
			s      = new(State)
		)

		s.Decode(state)
		if s.RedirectURI == "" {
			err := errors.New("invalid state with missing redirect_uri")
			slog.Warn("authorization code callback wrong state", httputil.SlogError(err))
			httputil.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if grantError := params.Get("error"); grantError != "" {
			slog.Warn("authorization code callback received error",
				slog.String("error", grantError),
			)
			redirectCallback(w, r, s.RedirectURI, false, state)
			return
		}

		config, err := model.TokenGetterConfig[oauth.AuthorizationCode](getter())
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError,
				httputil.SlogError(err),
				slog.String("client-id", s.ClientID),
				slog.String("user-id", s.UserID))
			return
		}

		token, err := config.Exchange(s.UserID, s.ClientID, params.Get("code"))
		if err != nil {
			slog.Error("authorization code callback token exchange failed",
				httputil.SlogError(err),
				slog.String("client-id", s.ClientID),
				slog.String("user-id", s.UserID),
			)
			redirectCallback(w, r, s.RedirectURI, false, state)
			return
		}

		var OIDCIdentity = token.Attrs.Coalesce(oauth.IDClaims...)

		_, err = repo.Users.Create(s.UserID, s.ClientID, OIDCIdentity, "")
		if err != nil {
			slog.Error("authorization code callback failed to create user",
				slog.String("oidc-identity", OIDCIdentity),
				slog.String("client-id", s.ClientID),
				slog.String("user-id", s.UserID),
				httputil.SlogError(err),
			)
			redirectCallback(w, r, s.RedirectURI, false, state)
			return
		}

		slog.Info("authorization code callback complete",
			slog.String("client-id", s.ClientID),
			slog.String("user-id", s.UserID),
		)
		redirectCallback(w, r, s.RedirectURI, true, state)
	}
}

func redirectCallback(w http.ResponseWriter, r *http.Request, uri string, success bool, state string) {
	redirect, err := url.Parse(uri)
	if err != nil {
		httputil.Error(w, "failed to parse middleware redirect", http.StatusInternalServerError,
			slog.String("redirect-uri", uri),
			slog.String("state", state),
		)
		return
	}

	q := redirect.Query()
	q.Add("state", state)
	q.Add("success", strconv.FormatBool(success))

	redirect.RawQuery = q.Encode()

	http.Redirect(w, r, redirect.String(), http.StatusFound)
}
