package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gitlab-user-synchronizer/internal/httpclient"
)

// kcGroup и kcUser — части API Keycloak, используемые в синхронизации.
type kcGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type kcUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Enabled   bool   `json:"enabled"`
}

// FetchGroupMembers возвращает всех пользователей указанной группы Keycloak.
func (kc *Client) FetchGroupMembers(ctx context.Context, token, groupName string) ([]kcUser, error) {
	groupID, err := kc.findGroupID(ctx, token, groupName)
	if err != nil {
		return nil, fmt.Errorf("resolve group %q id: %w", groupName, err)
	}

	membersURL := fmt.Sprintf("%s/admin/realms/%s/groups/%s/members?briefRepresentation=false&max=1000",
		strings.TrimRight(kc.cfg.KeycloakBaseURL, "/"),
		url.PathEscape(kc.cfg.KeycloakRealm),
		groupID)

	request, _ := http.NewRequest(http.MethodGet, membersURL, nil)
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := httpclient.DoWithRetry(ctx, kc.http, request)
	if err != nil {
		return nil, fmt.Errorf("members fetch: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("members HTTP %d: %s", response.StatusCode, body)
	}

	var users []kcUser
	if err = json.NewDecoder(response.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("decode members: %w", err)
	}

	return users, nil
}

// findGroupID ищет группу по имени и возвращает её ID.
func (kc *Client) findGroupID(ctx context.Context, token, name string) (string, error) {
	searchURL := fmt.Sprintf("%s/admin/realms/%s/groups?briefRepresentation=true&search=%s",
		strings.TrimRight(kc.cfg.KeycloakBaseURL, "/"),
		url.PathEscape(kc.cfg.KeycloakRealm),
		url.QueryEscape(name))

	request, _ := http.NewRequest(http.MethodGet, searchURL, nil)
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := httpclient.DoWithRetry(ctx, kc.http, request)
	if err != nil {
		return "", fmt.Errorf("groups search: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("group search HTTP %d: %s", response.StatusCode, body)
	}

	var groupList []kcGroup
	if err = json.NewDecoder(response.Body).Decode(&groupList); err != nil {
		return "", fmt.Errorf("decode group list: %w", err)
	}

	for _, group := range groupList {
		if group.Name == name {
			return group.ID, nil
		}
	}

	return "", fmt.Errorf("group %q not found", name)
}
