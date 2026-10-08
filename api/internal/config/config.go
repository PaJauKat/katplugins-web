package config

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Config contiene la configuracion del backend, leida desde variables de entorno.
type Config struct {
	SupabaseURL        string
	SupabaseSecretKey  string
	GoogleClientID     string
	GoogleClientSecret string
	AppBaseURL         string // URL publica base (para redirect_uri y post-login)
	SessionSecret      []byte
	PluginsBaseURL     string // base URL de los jars en R2 (ej: https://bucket.pajau.cl)
	SyncToken          string // token para el build (sync de plugins)
	AdminEmails        map[string]bool
	AllowedOrigins     []string
}

// Load lee la configuracion de entorno. Si existe un archivo .env (en el
// directorio actual o en el padre) se cargan sus valores sin sobreescribir
// variables ya presentes en el entorno.
func Load() *Config {
	loadDotEnv()

	appBase := strings.TrimRight(firstNonEmpty(os.Getenv("APP_BASE_URL"), "https://katplugins.pajau.cl"), "/")
	pluginsBase := strings.TrimRight(firstNonEmpty(os.Getenv("PLUGINS_BASE_URL"), "https://bucket.pajau.cl"), "/")

	return &Config{
		SupabaseURL:        strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabaseSecretKey:  firstNonEmpty(os.Getenv("SUPABASE_SECRET_KEY"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY")),
		GoogleClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		AppBaseURL:         appBase,
		SessionSecret:      loadSessionSecret(),
		PluginsBaseURL:     pluginsBase,
		SyncToken:          strings.TrimSpace(os.Getenv("SYNC_TOKEN")),
		AdminEmails:        parseEmails(os.Getenv("ADMIN_EMAILS")),
		AllowedOrigins:     splitCSV(os.Getenv("ALLOWED_ORIGINS")),
	}
}

// RedirectURL es el redirect_uri registrado en Google Cloud.
func (c *Config) RedirectURL() string {
	return c.AppBaseURL + "/auth/callback"
}

// CookieSecure indica si las cookies deben llevar el flag Secure.
func (c *Config) CookieSecure() bool {
	return strings.HasPrefix(c.AppBaseURL, "https://")
}

// GoogleEnabled indica si el login con Google esta configurado.
func (c *Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// IsAdminEmail indica si el correo pertenece a un admin definido por entorno.
func (c *Config) IsAdminEmail(email string) bool {
	if email == "" {
		return false
	}
	return c.AdminEmails[strings.ToLower(strings.TrimSpace(email))]
}

// IsOriginAllowed indica si un origen puede recibir CORS / ejecutar POST.
func (c *Config) IsOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if strings.EqualFold(origin, c.AppBaseURL) {
		return true
	}
	for _, o := range c.AllowedOrigins {
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}

func loadSessionSecret() []byte {
	if raw := strings.TrimSpace(os.Getenv("SESSION_SECRET")); raw != "" {
		return []byte(raw)
	}
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	secret := base64.RawURLEncoding.EncodeToString(buf)
	log.Println("ADVERTENCIA: SESSION_SECRET no definido; se genero uno temporal. " +
		"Define SESSION_SECRET en produccion o las sesiones se invalidaran al reiniciar.")
	return []byte(secret)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseEmails(raw string) map[string]bool {
	out := map[string]bool{}
	for _, e := range splitCSV(raw) {
		out[strings.ToLower(e)] = true
	}
	return out
}

func splitCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// loadDotEnv busca un archivo .env en el directorio actual y en el padre.
func loadDotEnv() {
	candidates := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "..", ".env"))
	}
	for _, path := range candidates {
		if f, err := os.Open(path); err == nil {
			parseDotEnv(f)
			f.Close()
			return
		}
	}
}

func parseDotEnv(f *os.File) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
