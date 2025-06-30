package syncer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode"

	"gitlab-user-synchronizer/internal/config"
	"gitlab-user-synchronizer/internal/gitlab"
	"gitlab-user-synchronizer/internal/keycloak"
)

// Run — главный цикл синхронизации «тенант → пользователи → GitLab».
func Run(
	ctx context.Context,
	cfg config.Config,
	kcClient *keycloak.Client,
	glClient *gitlab.Client,
	kcToken string,
) error {
	for _, tenant := range cfg.Tenants {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Printf("INFO: processing tenant %q", tenant)

		// 1. Получаем пользователей из Keycloak-группы.
		users, err := kcClient.FetchGroupMembers(ctx, kcToken, tenant)
		if err != nil {
			log.Printf("ERROR: fetch KC group %q: %v", tenant, err)
			continue
		}
		log.Printf("INFO: %d users in KC group %q", len(users), tenant)

		// 2. Для каждого пользователя проверяем/создаём GitLab-аккаунт.
		for _, kcUser := range users {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if kcUser.Email == "" {
				log.Printf("WARN: KC user %q missing email, skip", kcUser.Username)
				continue
			}

			gitUser, err := glClient.FindUserByEmail(ctx, kcUser.Email)
			if err != nil {
				log.Printf("ERROR: search GitLab user %q: %v", kcUser.Email, err)
				continue
			}

			if gitUser == nil {
				displayName := strings.TrimSpace(kcUser.FirstName + " " + kcUser.LastName)
				if displayName == "" {
					displayName = kcUser.Username
				}
				safeUsername := sanitizeUsername(kcUser.Username)

				gitUser, err = glClient.CreateUser(ctx, kcUser.Email, safeUsername, displayName)
				if err != nil {
					log.Printf("ERROR: create GitLab user %q: %v", kcUser.Email, err)
					continue
				}
			}

			// 3. Убеждаемся, что у пользователя есть доступ к группе-тенанту.
			groupPath := fmt.Sprintf("%s/%s", cfg.GitLabRootGroup, tenant)
			if err = glClient.EnsureGroupAccess(
				ctx, gitUser.ID, kcUser.Email, groupPath, cfg.GitLabUserRole, cfg.GitLabOverrideRole,
			); err != nil {
				log.Printf("ERROR: access for %q: %v", kcUser.Email, err)
				continue
			}
		}
	}

	return nil
}

// sanitizeUsername заменяет «плохие» символы на «_» и приводит к нижнему регистру.
func sanitizeUsername(raw string) string {
	var sb strings.Builder
	for _, symbol := range raw {
		if unicode.IsLetter(symbol) || unicode.IsDigit(symbol) || symbol == '-' || symbol == '_' || symbol == '.' {
			sb.WriteRune(unicode.ToLower(symbol))
		} else {
			sb.WriteRune('_')
		}
	}
	if out := sb.String(); out != "" {
		return out
	}

	return "user"
}
