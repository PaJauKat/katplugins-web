package server

import (
	"net/http"

	"github.com/PaJauKat/katplugins-web/api/internal/auth"
	"github.com/PaJauKat/katplugins-web/api/internal/config"
	"github.com/PaJauKat/katplugins-web/api/internal/supabase"
)

// Server agrupa la configuracion, el cliente de base de datos y los helpers
// de autenticacion, y expone el router HTTP.
type Server struct {
	cfg    *config.Config
	db     *supabase.Client
	signer *auth.Signer
	google *auth.GoogleOAuth
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg:    cfg,
		db:     supabase.New(cfg.SupabaseURL, cfg.SupabaseSecretKey),
		signer: auth.NewSigner(cfg.SessionSecret),
		google: auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.RedirectURL()),
	}
}

// Handler construye el multiplexor con todas las rutas de la API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Autenticacion (Google OAuth directo)
	mux.HandleFunc("GET /api/auth/google/login", s.handleGoogleLogin)
	mux.HandleFunc("GET /api/auth/desktop", s.handleDesktopLogin)
	mux.HandleFunc("GET /auth/callback", s.handleGoogleCallback)
	mux.HandleFunc("GET /api/auth/verify", s.handleVerify)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/plugins", s.handlePlugins)
	mux.HandleFunc("GET /api/manifest", s.handleManifest)
	mux.HandleFunc("GET /api/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("GET /api/entitlements", s.requireAuth(s.handleEntitlements))

	// Facturacion / pasarelas de pago
	mux.HandleFunc("GET /api/billing/providers", s.handleBillingProviders)
	mux.HandleFunc("POST /api/billing/checkout", s.requireAuth(s.handleBillingCheckout))
	mux.HandleFunc("GET /api/billing/subscription", s.requireAuth(s.handleBillingSubscription))
	mux.HandleFunc("GET /api/billing/flow/return", s.handleFlowReturn)
	mux.HandleFunc("POST /api/billing/flow/return", s.handleFlowReturn)
	mux.HandleFunc("POST /api/webhooks/flow", s.handleFlowWebhook)
	mux.HandleFunc("POST /api/webhooks/lemonsqueezy", s.handleLemonWebhook)
	mux.HandleFunc("POST /api/webhooks/nowpayments", s.handleNowPaymentsWebhook)

	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleAdminListUsers))
	mux.HandleFunc("POST /api/admin/users/tier", s.requireAdmin(s.handleAdminSetTier))
	mux.HandleFunc("GET /api/admin/users/access", s.requireAdmin(s.handleAdminUserAccess))
	mux.HandleFunc("POST /api/admin/access", s.requireAdmin(s.handleAdminSetAccess))

	mux.HandleFunc("GET /api/admin/plugins", s.requireAdmin(s.handleAdminListPlugins))
	mux.HandleFunc("POST /api/admin/plugins", s.requireAdmin(s.handleAdminUpsertPlugin))
	mux.HandleFunc("POST /api/admin/plugins/sync", s.handleAdminSyncPlugins)
	mux.HandleFunc("DELETE /api/admin/plugins", s.requireAdmin(s.handleAdminDeletePlugin))

	return s.withMiddleware(mux)
}
