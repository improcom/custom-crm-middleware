package oauth

import (
	"context"
	"crm-middleware/model"
	"crm-middleware/repo"
	"fmt"
	"log/slog"

	"golang.org/x/oauth2/clientcredentials"
)

type ClientCredentials struct {
	*clientcredentials.Config
	*ExtraConfig
}

func (config *ClientCredentials) Token(userID string, clientID string) (*model.Token, error) {
	t, err := repo.Tokens.Get(userID, clientID)
	if err != nil || t == nil || !oauth2Token(t).Valid() {
		slog.Debug("fetching new client credentials token", slog.String("user", userID), slog.String("client", clientID))

		token, err := config.Config.Token(context.Background())
		if err != nil {
			return nil, err
		}

		t = newToken(token, config.ExtraConfig)

		if err := repo.Tokens.Save(userID, clientID, t); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}

	return t, nil
}
