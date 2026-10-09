package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PaJauKat/katplugins-web/api/internal/auth"
)

// GET /api/auth/google/login — inicia el flujo OAuth web (navegador).
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.GoogleEnabled() {
		writeError(w, http.StatusServiceUnavailable, "el login con Google no está configurado")
		return
	}

	payload, challenge, err := s.newOAuthState(auth.OAuthState{
		Next: sanitizeNext(r.URL.Query().Get("next")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar el login")
		return
	}
	s.setOAuthCookie(w, s.mustSign(payload))
	http.Redirect(w, r, s.google.AuthURL(payload.State, challenge), http.StatusFound)
}

// GET /api/auth/desktop?port=8888&state=... — login desde el cliente de escritorio.
// Al terminar redirige a http://localhost:<port>/callback con el token de sesion.
func (s *Server) handleDesktopLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.GoogleEnabled() {
		writeError(w, http.StatusServiceUnavailable, "el login con Google no está configurado")
		return
	}

	port, err := strconv.Atoi(r.URL.Query().Get("port"))
	if err != nil || port < 1024 || port > 65535 {
		writeError(w, http.StatusBadRequest, "puerto inválido")
		return
	}

	payload, challenge, err := s.newOAuthState(auth.OAuthState{
		Next:     "/dashboard",
		Kind:     "desktop",
		Port:     port,
		CLIState: r.URL.Query().Get("state"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo iniciar el login")
		return
	}
	s.setOAuthCookie(w, s.mustSign(payload))
	http.Redirect(w, r, s.google.AuthURL(payload.State, challenge), http.StatusFound)
}

func (s *Server) newOAuthState(base auth.OAuthState) (auth.OAuthState, string, error) {
	state, err := auth.RandomState()
	if err != nil {
		return auth.OAuthState{}, "", err
	}
	verifier, challenge, err := auth.PKCEVerifier()
	if err != nil {
		return auth.OAuthState{}, "", err
	}
	base.State = state
	base.Verifier = verifier
	base.Exp = time.Now().Add(10 * time.Minute).Unix()
	return base, challenge, nil
}

func (s *Server) mustSign(v interface{}) string {
	token, err := s.signer.Sign(v)
	if err != nil {
		// Sign solo falla si el payload no es serializable; no debería ocurrir.
		log.Printf("error firmando token: %v", err)
	}
	return token
}

// GET /auth/callback — Google devuelve aquí el code.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("error") != "" {
		http.Redirect(w, r, "/login?error=google", http.StatusFound)
		return
	}

	cookie, err := r.Cookie(auth.OAuthCookieName)
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}
	var payload auth.OAuthState
	if err := s.signer.Verify(cookie.Value, &payload); err != nil {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}
	s.clearOAuthCookie(w)

	if payload.Exp < time.Now().Unix() || payload.State != q.Get("state") {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}

	code := q.Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?error=google", http.StatusFound)
		return
	}

	gu, err := s.google.Exchange(r.Context(), code, payload.Verifier)
	if err != nil {
		log.Printf("oauth: fallo el intercambio con Google: %v", err)
		http.Redirect(w, r, "/login?error=token", http.StatusFound)
		return
	}
	if gu.Email == "" {
		http.Redirect(w, r, "/login?error=email", http.StatusFound)
		return
	}

	profile, err := s.findOrCreateGoogleUser(r.Context(), gu)
	if err != nil {
		log.Printf("oauth: no se pudo crear/actualizar el perfil de %s: %v", gu.Email, err)
		http.Redirect(w, r, "/login?error=db", http.StatusFound)
		return
	}

	now := time.Now()
	session := auth.Session{
		UserID: profile.ID,
		Email:  profile.Email,
		Iat:    now.Unix(),
		Exp:    now.Add(maxSessionAge).Unix(),
	}
	token := s.mustSign(session)

	// Cliente de escritorio: devolver el token al loopback del manager.
	if payload.Kind == "desktop" {
		redirect := fmt.Sprintf(
			"http://localhost:%d/callback?status=success&token=%s&state=%s",
			payload.Port, url.QueryEscape(token), url.QueryEscape(payload.CLIState),
		)
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}

	s.setSessionCookie(w, token)
	http.Redirect(w, r, sanitizeNext(payload.Next), http.StatusFound)
}

// GET /api/auth/verify — valida un token de sesion (lo usa el loader/manager).
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := bearerOrCookie(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "falta el token")
		return
	}
	st := s.userFromToken(r, token)
	if st == nil {
		writeError(w, http.StatusUnauthorized, "token inválido")
		return
	}
	var sess auth.Session
	_ = s.signer.Verify(token, &sess)

	username := st.Profile.FullName
	if username == "" {
		username = st.Profile.Email
	}
	st.Profile.Tier = s.reconcileTier(r.Context(), st.Profile)
	writeJSON(w, http.StatusOK, map[string]any{
		"valid": true,
		"userData": map[string]any{
			"id":        st.Profile.ID,
			"username":  username,
			"email":     st.Profile.Email,
			"tier":      st.Profile.Tier,
			"isAdmin":   st.IsAdmin,
			"discordId": st.Profile.ID,
			"iat":       sess.Iat,
			"exp":       sess.Exp,
		},
	})
}

// POST /api/auth/logout — limpia la sesion.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func bearerOrCookie(r *http.Request) string {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}

// sanitizeNext evita open redirects: solo acepta rutas internas.
func sanitizeNext(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") ||
		strings.HasPrefix(next, "//") || strings.Contains(next, ":") {
		return "/dashboard"
	}
	return next
}
