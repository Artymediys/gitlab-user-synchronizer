package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// glUser — часть ответа GitLab API /users
type glUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// FindUserByEmail ищет пользователя по email
func (gl *Client) FindUserByEmail(ctx context.Context, email string) (*glUser, error) {
	searchURL := fmt.Sprintf("%s/api/v4/users?search=%s&per_page=100",
		strings.TrimRight(gl.cfg.GitLabBaseURL, "/"),
		url.QueryEscape(email))

	response, err := gl.do(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("user search: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("user search HTTP %d: %s", response.StatusCode, body)
	}

	var userList []glUser
	if err = json.NewDecoder(response.Body).Decode(&userList); err != nil {
		return nil, fmt.Errorf("decode user list: %w", err)
	}

	for _, user := range userList {
		if strings.EqualFold(user.Email, email) {
			return &user, nil
		}
	}

	return nil, nil
}

// CreateUser создаёт нового пользователя в GitLab
func (gl *Client) CreateUser(ctx context.Context, email, username, name string) (*glUser, error) {
	payload := fmt.Sprintf(`{"email":"%s","username":"%s","name":"%s","skip_confirmation":true,"force_random_password":true}`,
		email, username, name)

	response, err := gl.do(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v4/users", strings.TrimRight(gl.cfg.GitLabBaseURL, "/")),
		strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusCreated:
		var user glUser
		if err = json.NewDecoder(response.Body).Decode(&user); err != nil {
			return nil, fmt.Errorf("decode created user: %w", err)
		}
		log.Printf("INFO: GitLab user %q created (id=%d)", email, user.ID)
		return &user, nil
	case http.StatusConflict:
		return nil, fmt.Errorf("user already exists (409)")
	default:
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("create user HTTP %d: %s", response.StatusCode, body)
	}
}

// EnsureGroupAccess убеждается, что у пользователя есть нужная роль в группе
func (gl *Client) EnsureGroupAccess(
	ctx context.Context,
	userID int,
	email string,
	groupPath string,
	access int,
	override bool,
) error {
	escapedGroup := url.PathEscape(groupPath) // «instance/durs» → «instance%2Fdurs»

	memberURL := fmt.Sprintf("%s/api/v4/groups/%s/members/%d",
		strings.TrimRight(gl.cfg.GitLabBaseURL, "/"), escapedGroup, userID)

	// 1. Проверяем, есть ли уже членство
	membershipResponse, err := gl.do(ctx, http.MethodGet, memberURL, nil)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	defer membershipResponse.Body.Close()

	if membershipResponse.StatusCode == http.StatusOK {
		if !override {
			return nil
		}

		var current struct {
			AccessLevel int `json:"access_level"`
		}
		if err = json.NewDecoder(membershipResponse.Body).Decode(&current); err != nil {
			return fmt.Errorf("decode member: %w", err)
		}

		if current.AccessLevel == access {
			return nil
		}

		// Обновляем роль
		payload := fmt.Sprintf(`{"access_level":%d}`, access)
		roleResponse, err := gl.do(ctx, http.MethodPut, memberURL, strings.NewReader(payload))
		if err != nil {
			return fmt.Errorf("update member: %w", err)
		}
		defer roleResponse.Body.Close()

		if roleResponse.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(roleResponse.Body)
			return fmt.Errorf("update member HTTP %d: %s", roleResponse.StatusCode, body)
		}

		log.Printf("INFO: role updated for user %q (id=%d) in %s → roleID=%d", email, userID, groupPath, access)
		return nil
	}

	if membershipResponse.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(membershipResponse.Body)
		return fmt.Errorf("check member HTTP %d: %s", membershipResponse.StatusCode, body)
	}

	// 2. Добавляем нового участника
	addURL := fmt.Sprintf("%s/api/v4/groups/%s/members",
		strings.TrimRight(gl.cfg.GitLabBaseURL, "/"), escapedGroup)
	payload := fmt.Sprintf(`{"user_id":%d,"access_level":%d}`, userID, access)

	addResponse, err := gl.do(ctx, http.MethodPost, addURL, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	defer addResponse.Body.Close()

	if addResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(addResponse.Body)
		return fmt.Errorf("add member HTTP %d: %s", addResponse.StatusCode, body)
	}

	log.Printf("INFO: added user %q (id=%d) to %s with role id=%d", email, userID, groupPath, access)
	return nil
}
