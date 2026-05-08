package api

import (
	"crm-middleware/httputil"
	"crm-middleware/model"
	"crm-middleware/model/dataschema"
	"crm-middleware/oauth"
	"crm-middleware/repo"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
)

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		httputil.Error(w, "invalid authorization header with client id and client secret", http.StatusBadRequest)
		return
	}

	client, err := repo.Clients.Get(clientID)
	if err != nil || subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		httputil.Error(w, "invalid credentials", http.StatusForbidden,
			slog.String("client-id", clientID),
		)
		return
	}

	accessToken, err := httputil.IssueAccessToken(r.Header.Get("X-Integration-ID"), clientID)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusInternalServerError, slog.String("client-id", clientID))
		return
	}

	slog.Debug("access token issued", slog.String("client-id", clientID))
	httputil.JSONResponse(w, map[string]string{"token": accessToken})
}

func configHandler(config IntegrationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		client, err := repo.Clients.Get(claims.ClientID)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusBadRequest, claims.SlogAttr())
			return
		}

		if client.Status == model.ClientStatusActive {
			httputil.Error(w, "client already activated, configuration not allowed", http.StatusConflict, claims.SlogAttr())
			return
		}

		httputil.JSONResponse(w, config)
	}
}

func versionsHandlerFunc(appName string, versions ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if err := repo.Clients.UpdateStatus(claims.ClientID, appName, model.ClientStatusActive); err != nil {
			httputil.Error(w, err.Error(), http.StatusBadRequest, claims.SlogAttr())
			return
		}

		slog.Info("client activated", claims.SlogAttr())
		httputil.JSONResponse(w, versions)
	}
}

func objectTypesHandlerFunc(models ...*dataschema.Model) http.HandlerFunc {
	schemas := make([]*model.DataSchema, len(models))

	for i, m := range models {
		fields := make([]*model.DataSchemaField, 0, len(m.Fields))
		for _, f := range m.Fields {
			fields = append(fields, &model.DataSchemaField{
				ID:    f.ID,
				Type:  string(f.Type),
				Logic: string(f.Logic),
			})
		}
		schemas[i] = &model.DataSchema{ObjectType: m.ID, Fields: fields}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if err := repo.DataSchemas.Save(claims.ClientID, schemas...); err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		objectTypes := make([]string, 0, len(schemas))
		for _, s := range schemas {
			objectTypes = append(objectTypes, s.ObjectType)
		}

		slog.Debug("object types registered", claims.SlogAttr(), slog.Int("count", len(objectTypes)))
		httputil.JSONResponse(w, objectTypes)
	}
}

func describeObjectHandlerFunc(descriptions map[string]*dataschema.Model) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		obj := GetObject(r)
		model, ok := descriptions[obj.ObjectType]
		if !ok {
			httputil.Error(w, "unknown object type", http.StatusNotFound, claims.SlogAttr(), obj.SlogAttr())
			return
		}
		httputil.JSONResponse(w, model)
	}
}

func clientDeletedHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)

	slog.Info("client data deletion requested", claims.SlogAttr())

	if err := repo.Clients.DeleteClientData(claims.ClientID); err != nil {
		httputil.Error(w, "failed to delete client data", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	slog.Info("client data deleted", claims.SlogAttr())
	httputil.JSONResponse(w, struct{}{})
}

func updateCustomFieldsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		byObjectType map[string][]*dataschema.Field
		claims       = GetClaims(r)
	)

	if err := json.NewDecoder(r.Body).Decode(&byObjectType); err != nil {
		httputil.Error(w, "custom-fields: JSON decode error", http.StatusBadRequest,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	schemas := make([]*model.DataSchema, 0, len(byObjectType))
	for objectType, fields := range byObjectType {
		schema, err := repo.DataSchemas.Load(objectType, claims.ClientID, repo.DataSchemaExcludeCustom)
		if err != nil {
			httputil.Error(w, "custom-fields: failed to load schema", http.StatusInternalServerError,
				httputil.SlogError(err),
				claims.SlogAttr(),
			)
			return
		}

		for _, f := range fields {
			schema.Fields = append(schema.Fields, &model.DataSchemaField{
				ID:     f.ID,
				Type:   string(f.Type),
				Logic:  string(f.Logic),
				Custom: true,
			})
		}
		schemas = append(schemas, schema)
	}

	if err := repo.DataSchemas.Save(claims.ClientID, schemas...); err != nil {
		httputil.Error(w, "custom-fields: failed to save fields", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}
	slog.Debug("custom fields updated", claims.SlogAttr(), slog.Int("models", len(byObjectType)))
}

func OAuth2LoginURLHandlerFunc(getter func() model.TokenGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)

		slog.Info("generating OAuth login URL", claims.SlogAttr())

		config, err := model.TokenGetterConfig[oauth.AuthorizationCode](getter())
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError, claims.SlogAttr())
			return
		}

		redirectURI := r.URL.Query().Get("redirect_uri")
		if redirectURI == "" {
			httputil.Error(w, "required redirect_uri to origin is empty", http.StatusInternalServerError, claims.SlogAttr())
			return
		}

		state, err := State{
			OriginalState: r.URL.Query().Get("state"),
			ClientID:      claims.ClientID,
			UserID:        claims.UserID,
			RedirectURI:   redirectURI,
		}.Encode()

		if err != nil {
			httputil.Error(w, "failed to encode authorization code url state", http.StatusInternalServerError,
				httputil.SlogError(err),
				claims.SlogAttr(),
			)
			return
		}

		httputil.JSONResponse(w, map[string]string{
			"login_url": config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce),
		})
	}
}

func OAuth2PasswordLoginHandlerFunc(getter func() model.TokenGetter) http.HandlerFunc {
	type authResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims.UserID == "" {
			slog.Warn("password login: missing user_id in token claims", claims.SlogAttr())
			httputil.JSONResponse(w, authResponse{Status: "error", Message: "missing user_id"}, http.StatusBadRequest)
			return
		}

		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
			slog.Warn("password login: failed to decode credentials", httputil.SlogError(err), claims.SlogAttr())
			httputil.JSONResponse(w, authResponse{Status: "error", Message: "invalid request body"}, http.StatusBadRequest)
			return
		}

		config, err := model.TokenGetterConfig[oauth.Password](getter())
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError, claims.SlogAttr())
			return
		}

		_, err = config.PasswordCredentialsToken(claims.UserID, claims.ClientID, credentials.Username, credentials.Password)
		if err != nil {
			slog.Error("OAuth password login failed", httputil.SlogError(err), claims.SlogAttr())
			httputil.JSONResponse(w, authResponse{Status: "error", Message: "OAuth password: token request failed"}, http.StatusUnauthorized)
			return
		}

		if _, err = repo.Users.Create(claims.UserID, claims.ClientID, credentials.Username, credentials.Password); err != nil {
			slog.Error("failed to create user", slog.String("user", credentials.Username), httputil.SlogError(err))
			httputil.JSONResponse(w, authResponse{Status: "error", Message: "failed to create user"}, http.StatusInternalServerError)
			return
		}

		slog.Info("OAuth authorization complete",
			claims.SlogAttr(),
			slog.String("user", credentials.Username),
		)
		httputil.JSONResponse(w, authResponse{Status: "ok", Message: "authorized"})
	}
}

func OAuth2StatusHandlerFunc(getter func() model.TokenGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		status := struct {
			Authorized bool   `json:"authorized"`
			Email      string `json:"email"`
		}{Authorized: true}

		token, err := getter().Token(claims.UserID, claims.ClientID)
		if err != nil {
			status.Authorized = false
			slog.Warn("user unauthorized", httputil.SlogError(err), claims.SlogAttr())
		}

		if token != nil {
			status.Email = token.Attrs.Coalesce(oauth.IDClaims...)
		}

		httputil.JSONResponse(w, status)
	}
}

func Empty(_ http.ResponseWriter, _ *http.Request) {}

func UnauthorizedClientUser(w http.ResponseWriter, r *http.Request) {
	httputil.JSONResponse(w, AuthStatus{})
}

func AuthorizedClientUser(w http.ResponseWriter, _ *http.Request) {
	httputil.JSONResponse(w, AuthStatus{Authorized: true})
}

func DeleteClientUser(_ http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if err := repo.Users.Delete(claims.UserID, claims.ClientID); err != nil {
		slog.Warn("failed to delete user", claims.SlogAttr(), httputil.SlogError(err))
	} else {
		slog.Debug("client user deleted", claims.SlogAttr())
	}
}
