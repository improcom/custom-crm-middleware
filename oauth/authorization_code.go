package oauth

import (
	"context"
	"crm-middleware/model"
	"crm-middleware/repo"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/oauth2"
)

type AuthorizationCode struct {
	*oauth2.Config
	*ExtraConfig
}

func (config *AuthorizationCode) Token(userID string, clientID string) (t *model.Token, err error) {
	t, err = repo.Tokens.Get(userID, clientID)
	if err != nil {
		return nil, err
	}

	token := oauth2Token(t)
	if token.Valid() {
		return t, nil
	}

	freshToken, err := config.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token, please log in again: %w", err)
	}

	if freshToken.AccessToken != token.AccessToken {
		slog.Debug("oauth token refreshed", slog.String("user", userID), slog.String("client", clientID))

		t = newToken(freshToken, config.ExtraConfig)

		if err := repo.Tokens.Save(userID, clientID, t); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}

	return t, nil
}

func (config *AuthorizationCode) Exchange(userID string, clientID string, code string) (*model.Token, error) {
	if code == "" {
		return nil, errors.New("oauth callback received with empty code")
	}

	token, err := config.Config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("oauth callback code exchange: %w", err)
	}

	var t = newToken(token, config.ExtraConfig)

	if err := repo.Tokens.Save(userID, clientID, t); err != nil {
		return nil, fmt.Errorf("oauth callback save token: %w", err)
	}

	return t, nil
}
