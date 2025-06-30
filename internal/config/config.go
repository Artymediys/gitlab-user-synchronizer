package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config хранит все параметры, поступающие из переменных окружения.
type Config struct {
	// Тенант-группы (Keycloak ↔ GitLab sub-group).
	Tenants []string

	// Keycloak — OIDC-сервер-источник пользователей.
	KeycloakBaseURL  string
	KeycloakRealm    string
	KeycloakClientID string
	KeycloakSecret   string

	// GitLab — куда создаём пользователей и права.
	GitLabBaseURL   string
	GitLabToken     string
	GitLabRootGroup string

	// Access-level GitLab по умолчанию и режим override.
	GitLabUserRole     int
	GitLabOverrideRole bool
}

// Load читает переменные окружения и формирует Config.
func Load() (Config, error) {
	get := func(name string) string { return strings.TrimSpace(os.Getenv(name)) }

	cfg := Config{
		Tenants: splitList(get("TENANT_LIST")),

		KeycloakBaseURL:  get("KEYCLOAK_BASE_URL"),
		KeycloakRealm:    get("KEYCLOAK_REALM"),
		KeycloakClientID: get("KEYCLOAK_CLIENT_ID"),
		KeycloakSecret:   get("KEYCLOAK_CLIENT_SECRET"),

		GitLabBaseURL:   get("GITLAB_BASE_URL"),
		GitLabToken:     get("GITLAB_TOKEN"),
		GitLabRootGroup: strings.Trim(get("GITLAB_ROOT_GROUP"), "/"),

		GitLabUserRole:     parseRole(get("GITLAB_USER_ROLE")),
		GitLabOverrideRole: parseBool(get("GITLAB_OVERRIDE_ROLE")),
	}

	// Минимальная валидация.
	if cfg.KeycloakBaseURL == "" || cfg.KeycloakRealm == "" || cfg.KeycloakClientID == "" ||
		cfg.KeycloakSecret == "" || cfg.GitLabBaseURL == "" || cfg.GitLabToken == "" ||
		cfg.GitLabRootGroup == "" || len(cfg.Tenants) == 0 {
		return Config{}, fmt.Errorf("missing required environment variables")
	}

	return cfg, nil
}

// splitList парсит переданный список тенантов
func splitList(csv string) []string {
	raw := strings.Split(csv, ",")
	out := make([]string, 0, len(raw))

	for _, v := range raw {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}

// parseRole определяет какую роль задавать юзерам
func parseRole(v string) int {
	aliases := map[string]int{
		"guest": 10, "reporter": 20, "developer": 30,
		"maintainer": 40, "owner": 50,
	}

	// Возвращает roleID, если в конфиге указано текстовое обозначение роли
	if roleID, ok := aliases[strings.ToLower(strings.TrimSpace(v))]; ok {
		return roleID
	}

	// Возвращает roleID, если в конфиге указан id роли
	if roleID, err := strconv.Atoi(v); err == nil {
		return roleID
	}

	// Если в конфиге не указали роль, то по-дефолту выставляется роль developer
	return 30
}

// parseBool преобразует строку в bool
func parseBool(value string) bool {
	// Если поступил невалидный value, то вернётся false
	// Валидными считаются: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"
	boolResult, _ := strconv.ParseBool(value)
	return boolResult
}
