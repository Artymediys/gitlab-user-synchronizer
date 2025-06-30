package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"gitlab-user-synchronizer/internal/config"
	"gitlab-user-synchronizer/internal/httpclient"
)

// Client обращается к Keycloak Admin API напрямую.
type Client struct {
	cfg  config.Config
	http *http.Client
}

// NewClient создаёт Keycloak HTTP-клиент.
func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg:  cfg,
		http: httpclient.New(),
	}
}

// tokenResponse — JSON-ответ Keycloak на /token.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// FetchTokenWithRetry запрашивает access-token с повторами по сетевым ошибкам.
func (kc *Client) FetchTokenWithRetry(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimRight(kc.cfg.KeycloakBaseURL, "/"),
		url.PathEscape(kc.cfg.KeycloakRealm))

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {kc.cfg.KeycloakClientID},
		"client_secret": {kc.cfg.KeycloakSecret},
	}

	request, _ := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := httpclient.DoWithRetry(ctx, kc.http, request)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("token request HTTP %d: %s", response.StatusCode, body)
	}

	var tr tokenResponse
	if err = json.NewDecoder(response.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token JSON: %w", err)
	}

	log.Printf("INFO: Keycloak token OK, expires in %d s", tr.ExpiresIn)
	return tr.AccessToken, nil
}
