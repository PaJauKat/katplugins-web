package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Cookie names.
const (
	SessionCookieName = "kat_session"
	OAuthCookieName   = "kat_oauth"
)

var ErrInvalidToken = errors.New("token invalido")

// Signer firma y verifica tokens con HMAC-SHA256 (formato payload.sig en base64url).
type Signer struct {
	secret []byte
}

func NewSigner(secret []byte) *Signer {
	return &Signer{secret: secret}
}

// Sign serializa v a JSON y devuelve un token firmado.
func (s *Signer) Sign(v interface{}) (string, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.sign(encoded), nil
}

// Verify valida la firma y decodifica el payload en v.
func (s *Signer) Verify(token string, v interface{}) error {
	encoded, sig, found := strings.Cut(token, ".")
	if !found {
		return ErrInvalidToken
	}
	expected := s.sign(encoded)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return ErrInvalidToken
	}
	return json.Unmarshal(payload, v)
}

func (s *Signer) sign(encoded string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Session es la sesion firmada que viaja en la cookie.
type Session struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

func (s Session) Expired() bool {
	return s.Exp < time.Now().Unix()
}

// OAuthState es el estado temporal del flujo OAuth (cookie de corta vida).
type OAuthState struct {
	State    string `json:"state"`
	Verifier string `json:"verifier"`
	Next     string `json:"next"`
	Exp      int64  `json:"exp"`
	// Desktop: cuando el login lo inicia el cliente de escritorio.
	Kind     string `json:"kind,omitempty"` // "" (web) | "desktop"
	Port     int    `json:"port,omitempty"`
	CLIState string `json:"cli_state,omitempty"`
}
