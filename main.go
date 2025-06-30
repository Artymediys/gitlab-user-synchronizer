package main

import (
	"context"
	"log"
	"time"

	"gitlab-user-synchronizer/internal/config"
	"gitlab-user-synchronizer/internal/gitlab"
	"gitlab-user-synchronizer/internal/keycloak"
	"gitlab-user-synchronizer/internal/syncer"
)

func main() {
	// Загрузка конфигурации приложения
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL ERROR: load config: %v", err)
	}

	// Настройка базового набора информации при выводе логов
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Контекст на весь запуск Job
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()

	// Клиенты Keycloak и GitLab
	keycloakClient := keycloak.NewClient(cfg)
	gitLabClient := gitlab.NewClient(cfg)

	// Получение сервисного токена Keycloak
	kcToken, err := keycloakClient.FetchTokenWithRetry(ctx)
	if err != nil {
		log.Fatalf("FATAL ERROR: cannot obtain Keycloak token: %v", err)
	}

	// Запуск синхронизации
	if err = syncer.Run(ctx, cfg, keycloakClient, gitLabClient, kcToken); err != nil {
		log.Fatalf("FATAL ERROR: sync finished with error: %v", err)
	}

	log.Printf("INFO: sync completed successfully")
}
