package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleAuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	googleUserinfoURL   = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// GoogleUser es la informacion del usuario devuelta por Google.
type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// GoogleOAuth implementa el flujo Authorization Code + PKCE contra Google.
type GoogleOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	http         *http.Client
}

func NewGoogleOAuth(clientID, clientSecret, redirectURL string) *GoogleOAuth {
	return &GoogleOAuth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		http:         &http.Client{Timeout: 15 * time.Second},
	}
}

// AuthURL construye la URL de consentimiento de Google.
func (g *GoogleOAuth) AuthURL(state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", g.ClientID)
	q.Set("redirect_uri", g.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("access_type", "online")
	q.Set("prompt", "select_account")
	return googleAuthEndpoint + "?" + q.Encode()
}

// Exchange cambia el code por tokens y obtiene el perfil del usuario.
func (g *GoogleOAuth) Exchange(ctx context.Context, code, codeVerifier string) (*GoogleUser, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("redirect_uri", g.RedirectURL)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google token error %d: %s", resp.StatusCode, string(body))
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("google no devolvio access_token")
	}
	return g.fetchUserinfo(ctx, token.AccessToken)
}

func (g *GoogleOAuth) fetchUserinfo(ctx context.Context, accessToken string) (*GoogleUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google userinfo error %d: %s", resp.StatusCode, string(body))
	}

	var u GoogleUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// RandomState genera un valor aleatorio para el parametro state.
func RandomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// PKCEVerifier genera un code_verifier y su code_challenge (S256).
func PKCEVerifier() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}
