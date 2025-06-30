package httpclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// New возвращает настроенный http.Client с тайм-аутом 30 сек.
func New() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
}

// DoWithRetry выполняет req, повторяя до maxAttempts при сетевых / 5xx-ошибках.
func DoWithRetry(ctx context.Context, client *http.Client, request *http.Request) (*http.Response, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err := client.Do(request.WithContext(ctx))
		if err == nil && response.StatusCode < 500 {
			return response, nil // успех
		}

		// Логируем неудачную попытку.
		if err != nil {
			log.Printf("WARN: %s attempt %d failed: %v", request.URL, attempt, err)
			lastErr = fmt.Errorf("network error: %w", err)
		} else {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()

			log.Printf("WARN: %s attempt %d got HTTP %d: %s", request.URL, attempt, response.StatusCode, body)
			lastErr = errors.New(response.Status)
		}

		// Последняя попытка — больше не ждём.
		if attempt == maxAttempts {
			break
		}

		// Ожидаем минуту или до отмены контекста.
		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}
