package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"simas-backend/internal/config"
	"simas-backend/internal/model"
)

var validIssuers = []string{
	"accounts.google.com",
	"https://accounts.google.com",
}

type GoogleOAuth struct {
	clientIDs []string
}

func NewGoogleOAuth(cfg *config.OAuthConfig) *GoogleOAuth {
	ids := []string{}
	if cfg.GoogleClientID != "" {
		ids = append(ids, cfg.GoogleClientID)
	}
	if cfg.MobileGoogleClientID != "" {
		ids = append(ids, cfg.MobileGoogleClientID)
	}
	return &GoogleOAuth{clientIDs: ids}
}

func (g *GoogleOAuth) ValidateIDToken(idToken string) (*model.GoogleUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrTokenInvalid
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenInfo struct {
		Audience      string `json:"aud"`
		Issuer        string `json:"iss"`
		UserID        string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		ExpiresAt     string `json:"exp"`
		Error         string `json:"error"`
	}

	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, err
	}
	if tokenInfo.Error != "" {
		return nil, ErrTokenInvalid
	}

	// Validate issuer
	issuerOK := false
	for _, iss := range validIssuers {
		if tokenInfo.Issuer == iss {
			issuerOK = true
			break
		}
	}
	if !issuerOK {
		return nil, errors.New("invalid issuer")
	}

	// Validate audience
	audOK := false
	for _, id := range g.clientIDs {
		if tokenInfo.Audience == id {
			audOK = true
			break
		}
	}
	if !audOK {
		return nil, errors.New("invalid audience")
	}

	// Validate expiry
	exp, err := strconv.ParseInt(tokenInfo.ExpiresAt, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return nil, errors.New("token expired")
	}

	// Validate required fields
	if tokenInfo.UserID == "" || tokenInfo.Email == "" {
		return nil, ErrTokenInvalid
	}

	return &model.GoogleUserInfo{
		ID:            tokenInfo.UserID,
		Email:         tokenInfo.Email,
		VerifiedEmail: tokenInfo.EmailVerified == "true",
		Name:          tokenInfo.Name,
		Picture:       tokenInfo.Picture,
	}, nil
}

type GoogleTokenExchange struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthCode     string
}

func ExchangeCodeForTokens(code, clientID, clientSecret, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.Post("https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrTokenInvalid
	}

	var tokenResp struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.IDToken, nil
}

func NewGoogleOAuthTokenExpiry() time.Duration {
	return 24 * time.Hour
}

var _ = fmt.Sprintf // keep import if unused

