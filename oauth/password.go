package oauth

import (
	"context"
	"crm-middleware/model"
	"crm-middleware/repo"
	"fmt"
	"log/slog"

	"golang.org/x/oauth2"
)

type Password struct {
	*oauth2.Config
	*ExtraConfig
}

func (config *Password) Token(userID, clientID string) (*model.Token, error) {
	t, err := repo.Tokens.Get(userID, clientID)
	if err != nil {
		return nil, err
	}

	token := oauth2Token(t)
	if token.Valid() {
		return t, nil
	}

	if token.RefreshToken == "" {
		// Some providers treat password grant as Client Credentials with User Credentials.
		// In that case refresh token might not be issued, so need to exchange credentials.

		user, err := repo.Users.Get(userID, clientID)
		if err != nil {
			return nil, fmt.Errorf("session expired and stored credentials unavailable: %w", err)
		}

		return config.PasswordCredentialsToken(userID, clientID, user.Username, user.Password)
	}

	freshToken, err := config.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh password grant token, please log in again: %w", err)
	}

	if freshToken.AccessToken != token.AccessToken {
		slog.Debug("oauth token refresh", slog.String("user", userID), slog.String("client", clientID))

		t = newToken(freshToken, config.ExtraConfig)

		if err := repo.Tokens.Save(userID, clientID, t); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}

	return t, nil
}

func (config *Password) PasswordCredentialsToken(userID, clientID, username, password string) (*model.Token, error) {
	token, err := config.Config.PasswordCredentialsToken(context.Background(), username, password)
	if err != nil {
		return nil, err
	}

	var t = newToken(token, config.ExtraConfig)

	if err := repo.Tokens.Save(userID, clientID, t); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return t, nil
}
