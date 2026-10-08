package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/PaJauKat/katplugins-web/api/internal/auth"
	"github.com/PaJauKat/katplugins-web/api/internal/models"
)

type ctxKey string

const authCtxKey ctxKey = "auth"

// authState es el resultado de autenticar una request.
type authState struct {
	Profile *models.User
	IsAdmin bool
}

func stateFrom(ctx context.Context) *authState {
	s, _ := ctx.Value(authCtxKey).(*authState)
	return s
}

// requireAuth verifica la cookie de sesion y carga el perfil del usuario.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.db.Configured() {
			writeError(w, http.StatusServiceUnavailable, "backend no configurado (faltan variables de Supabase)")
			return
		}

		st := s.currentUser(r)
		if st == nil {
			writeError(w, http.StatusUnauthorized, "sesión no válida")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), authCtxKey, st)))
	}
}

// requireAdmin exige que el usuario autenticado sea administrador.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		st := stateFrom(r.Context())
		if st == nil || !st.IsAdmin {
			writeError(w, http.StatusForbidden, "requiere permisos de administrador")
			return
		}
		next(w, r)
	})
}

// currentUser valida la cookie de sesion (o un Bearer) y devuelve el usuario, o nil.
func (s *Server) currentUser(r *http.Request) *authState {
	if !s.db.Configured() {
		return nil
	}

	token := ""
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie.Value != "" {
		token = cookie.Value
	} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if token == "" {
		return nil
	}
	return s.userFromToken(r, token)
}

func (s *Server) userFromToken(r *http.Request, token string) *authState {
	var sess auth.Session
	if err := s.signer.Verify(token, &sess); err != nil {
		return nil
	}
	if sess.Expired() {
		return nil
	}

	profile, err := s.getProfile(r.Context(), sess.UserID)
	if err != nil || profile == nil {
		return nil
	}

	return &authState{
		Profile: profile,
		IsAdmin: profile.Role == "admin" || s.cfg.IsAdminEmail(profile.Email),
	}
}

// tryAuth intenta autenticar sin fallar; devuelve nil si no hay sesion valida.
func (s *Server) tryAuth(r *http.Request) *authState {
	return s.currentUser(r)
}

// withMiddleware aplica chequeo de origen (CSRF) para metodos mutables.
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.cfg.IsOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proteccion CSRF basica: los metodos mutables deben venir del mismo origen.
		switch r.Method {
		case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
			if origin != "" && !s.cfg.IsOriginAllowed(origin) {
				writeError(w, http.StatusForbidden, "origen no permitido")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// maxSessionAge es la duracion de la sesion (y de la cookie).
const maxSessionAge = 7 * 24 * time.Hour

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(maxSessionAge.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) setOAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.OAuthCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearOAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.OAuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
