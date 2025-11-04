package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrOAuth2NotConfigured = errors.New("OAuth2 non configuré")
	ErrInvalidState         = errors.New("état OAuth2 invalide")
)

// OAuth2Config contient la configuration OAuth2 pour Authentik
type OAuth2Config struct {
	Provider     string // "authentik"
	ClientID     string
	ClientSecret string
	BaseURL      string // URL de base d'Authentik (ex: https://auth.example.com)
	RedirectURL  string // URL de callback (ex: https://remoteshell.example.com/api/auth/callback)
	Scopes       []string
	config       *oauth2.Config
}

// UserInfo contient les informations utilisateur depuis Authentik
type UserInfo struct {
	Sub           string `json:"sub"`           // ID utilisateur
	Email         string `json:"email"`
	Name          string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Groups        []string `json:"groups"`
}

// NewOAuth2Config crée une nouvelle configuration OAuth2
func NewOAuth2Config(provider, clientID, clientSecret, baseURL, redirectURL string, scopes []string) *OAuth2Config {
	authURL := fmt.Sprintf("%s/application/o/authorize/", baseURL)
	tokenURL := fmt.Sprintf("%s/application/o/token/", baseURL)

	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return &OAuth2Config{
		Provider:     provider,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      baseURL,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		config:       config,
	}
}

// GetAuthURL génère l'URL d'autorisation avec un state pour la sécurité
func (o *OAuth2Config) GetAuthURL(state string) string {
	if o.config == nil {
		return ""
	}
	return o.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode échange le code d'autorisation contre un token
func (o *OAuth2Config) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if o.config == nil {
		return nil, ErrOAuth2NotConfigured
	}
	return o.config.Exchange(ctx, code)
}

// GetUserInfo récupère les informations utilisateur depuis Authentik
func (o *OAuth2Config) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	if o.config == nil {
		return nil, ErrOAuth2NotConfigured
	}

	userInfoURL := fmt.Sprintf("%s/application/o/userinfo/", o.BaseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erreur de création de requête: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur de requête userinfo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erreur userinfo (status %d): %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("erreur de décodage userinfo: %v", err)
	}

	return &userInfo, nil
}

// GenerateState génère un state aléatoire pour la sécurité OAuth2
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateState valide le state (pour éviter les attaques CSRF)
func ValidateState(state, expectedState string) error {
	if state != expectedState {
		return ErrInvalidState
	}
	return nil
}

