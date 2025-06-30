package gitlab

import (
	"context"
	"io"
	"net/http"

	"gitlab-user-synchronizer/internal/config"
	"gitlab-user-synchronizer/internal/httpclient"
)

// Client выполняет вызовы GitLab REST API.
type Client struct {
	cfg  config.Config
	http *http.Client
}

// NewClient инициализирует GitLab API-клиент.
func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg:  cfg,
		http: httpclient.New(),
	}
}

// helper добавляет токен и делает GET/POST/PUT.
func (gl *Client) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	request, _ := http.NewRequest(method, endpoint, body)
	request.Header.Set("PRIVATE-TOKEN", gl.cfg.GitLabToken)

	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}

	return httpclient.DoWithRetry(ctx, gl.http, request)
}
