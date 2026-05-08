package salesforce

import (
	"bytes"
	"crm-middleware/api"
	"crm-middleware/config"
	"crm-middleware/logger"
	"crm-middleware/model"
	"crm-middleware/model/dataschema"
	"crm-middleware/oauth"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type salesforce struct {
	*api.AppV1
	httpClient  *http.Client
	tokenGetter func() model.TokenGetter
}

const (
	Version = "60.0"

	Account = "Account"
	Contact = "Contact"
	Lead    = "Lead"
	Task    = "Task"

	authURI  = "https://login.salesforce.com/services/oauth2/authorize"
	tokenURI = "https://login.salesforce.com/services/oauth2/token"
)

var (
	//go:embed icons/salesforce-user.webp
	userIcon []byte
	//go:embed icons/salesforce-keys.webp
	keysIcon []byte
	//go:embed icons/salesforce-chip.webp
	chipIcon []byte
)

type oauthConfigs struct {
	code        *oauth.AuthorizationCode
	password    *oauth.Password
	credentials *oauth.ClientCredentials
}

var (
	//go:embed schemas
	schemaFiles        embed.FS
	objectDescriptions = map[string]*dataschema.Model{
		Account: dataschema.MustLoad(schemaFiles, "schemas/"+Account+".json"),
		Contact: dataschema.MustLoad(schemaFiles, "schemas/"+Contact+".json"),
		Lead:    dataschema.MustLoad(schemaFiles, "schemas/"+Lead+".json"),
	}

	oAuthConfig atomic.Pointer[oauthConfigs]

	oAuthExtraConfig = &oauth.ExtraConfig{
		Expiry: 15 * time.Minute,
		Attrs:  oauth.AttrsCollector("email", "instance_url"),
	}
)

func App(authMethod api.AuthMethod) (sf *salesforce) {
	sf = &salesforce{AppV1: &api.AppV1{
		Path:     baseApiPattern(authMethod),
		Versions: []string{Version},
		Objects:  objectDescriptions,

		Config: api.IntegrationConfig{
			Name:       "Salesforce",
			AuthMethod: authMethod,
		},
	}}

	sf.AppV1.AuthHandlers = api.AuthHandlers{
		Status: api.UnauthorizedClientUser,
		Logout: api.DeleteClientUser,
	}

	sf.AppV1.ObjectHandlers = api.ObjectHandlers{
		Create: sf.ObjectCreateHandler,
		Get:    sf.ObjectGetHandler,
		Update: sf.ObjectUpdateHandler,
	}

	sf.AppV1.SearchHandlers = api.SearchHandlers{
		All:          sf.SearchHandler,
		Associations: sf.SearchAssociationsHandler,
	}

	sf.AppV1.TaskHandlers = api.TaskHandlers{
		Create: sf.CreateTaskHandler,
	}

	sf.httpClient = &http.Client{Timeout: 15 * time.Second}

	switch authMethod {
	case api.AuthMethodOAuth2:
		sf.tokenGetter = func() model.TokenGetter { return oAuthConfig.Load().code }
		sf.Config.ImageData = base64.StdEncoding.EncodeToString(userIcon)

		sf.AuthHandlers.Login = api.OAuth2LoginURLHandlerFunc(sf.tokenGetter)
		sf.AuthHandlers.Status = api.OAuth2StatusHandlerFunc(sf.tokenGetter)
		sf.AuthHandlers.Callback = api.AuthorizationCodeCallbackHandlerFunc(sf.tokenGetter)

	case api.AuthMethodBasicAuth:
		sf.tokenGetter = func() model.TokenGetter { return oAuthConfig.Load().password }
		sf.Config.ImageData = base64.StdEncoding.EncodeToString(keysIcon)

		sf.AuthHandlers.Login = api.OAuth2PasswordLoginHandlerFunc(sf.tokenGetter)
		sf.AuthHandlers.Status = api.OAuth2StatusHandlerFunc(sf.tokenGetter)

	case api.AuthMethodNone:
		sf.tokenGetter = func() model.TokenGetter { return oAuthConfig.Load().credentials }
		sf.Config.ImageData = base64.StdEncoding.EncodeToString(chipIcon)

		sf.AuthHandlers.Login = api.Empty
		sf.AuthHandlers.Status = api.AuthorizedClientUser
	}

	return sf
}

func authorizationCodeConfig(consumerKey, consumerSecret, callbackURL string) *oauth.AuthorizationCode {
	return &oauth.AuthorizationCode{
		Config: &oauth2.Config{
			ClientID:     consumerKey,
			ClientSecret: consumerSecret,
			RedirectURL:  callbackURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURI,
				TokenURL: tokenURI,
			},
			Scopes: []string{"openid", "email", "refresh_token", "offline_access", "api"},
		},
		ExtraConfig: oAuthExtraConfig,
	}
}

func passwordGrantConfig(consumerKey, consumerSecret string) *oauth.Password {
	return &oauth.Password{
		Config: &oauth2.Config{
			ClientID:     consumerKey,
			ClientSecret: consumerSecret,
			Endpoint:     oauth2.Endpoint{TokenURL: tokenURI},
		},
		ExtraConfig: oAuthExtraConfig,
	}
}

func clientCredentialsConfig(consumerKey, consumerSecret, domain string) *oauth.ClientCredentials {
	return &oauth.ClientCredentials{
		Config: &clientcredentials.Config{
			ClientID:     consumerKey,
			ClientSecret: consumerSecret,
			TokenURL:     fmt.Sprintf("https://%s/services/oauth2/token", domain),
		},
		ExtraConfig: oAuthExtraConfig,
	}
}

func RefreshOAuthConfig() {
	var (
		stderr   = logger.Stderr("salesforce")
		conf     = config.Get()
		host     = conf.Host.Scheme + "://" + conf.Host.Domain
		consumer = conf.Apps.Salesforce.Consumer
	)

	if consumer.Key == "" || consumer.Secret == "" {
		stderr("consumer key/secret not set -- edit config and reload")
	}

	if conf.Apps.Salesforce.Domain == "" {
		stderr("domain not set -- apps using client credentials require domain -- edit config and reload")
	}

	if conf.Host.Domain == "localhost" || conf.Host.Domain == "127.0.0.1" {
		host += ":" + fmt.Sprint(conf.Server.Port)
	}

	callbackURL := host + baseApiPattern(api.AuthMethodOAuth2) + api.CallbackEndpoint
	stderr("OAuth2 callback URL:", callbackURL)

	oAuthConfig.Store(&oauthConfigs{
		code:        authorizationCodeConfig(consumer.Key, consumer.Secret, callbackURL),
		password:    passwordGrantConfig(consumer.Key, consumer.Secret),
		credentials: clientCredentialsConfig(consumer.Key, consumer.Secret, conf.Apps.Salesforce.Domain),
	})
}

// Makes an authenticated request to the Salesforce API for the given user.
func (salesforce *salesforce) Request(clientID, userID, method, path string, data []byte) ([]byte, error) {
	auth, err := salesforce.tokenGetter().Token(userID, clientID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, auth.Attrs.Get("instance_url")+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", auth.Type+" "+auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("Salesforce API request", "method", method, "url", req.URL.String())

	resp, err := salesforce.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		slog.Warn("Salesforce API error", "status", resp.StatusCode, "method", method, "path", path, "body", string(body))
		return body, fmt.Errorf("Salesforce returned %d", resp.StatusCode)
	}
	return body, nil
}
