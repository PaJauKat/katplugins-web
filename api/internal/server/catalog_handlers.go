package server

import (
	"crypto/subtle"
	"net/http"
	"net/url"

	"github.com/PaJauKat/katplugins-web/api/internal/models"
)

// GET /api/entitlements — plugins habilitados para el usuario (web o escritorio).
func (s *Server) handleEntitlements(w http.ResponseWriter, r *http.Request) {
	st := stateFrom(r.Context())

	plugins, err := s.listPlugins(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar el catálogo")
		return
	}
	overrides, err := s.loadOverrides(r.Context(), st.Profile.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los accesos")
		return
	}

	st.Profile.Tier = s.reconcileTier(r.Context(), st.Profile)
	access := s.buildAccess(st.Profile.Tier, plugins, overrides)
	enabled := make([]models.Plugin, 0, len(access))
	for _, pa := range access {
		if pa.Access && pa.Plugin.URL != "" {
			enabled = append(enabled, pa.Plugin)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tier":    st.Profile.Tier,
		"plugins": enabled,
	})
}

// GET /api/manifest — catálogo publico con metadatos de empaquetado.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.listPlugins(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar el catálogo")
		return
	}
	out := make([]models.Plugin, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p.toModel(s.cfg.PluginsBaseURL))
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": out})
}

type syncPlugin struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Package      string `json:"package"`
	JarFile      string `json:"jarFile"`
	Version      string `json:"version"`
	Sha256       string `json:"sha256"`
	RequiredTier string `json:"requiredTier"`
}

type syncRequest struct {
	Plugins []syncPlugin `json:"plugins"`
}

// POST /api/admin/plugins/sync — registra/actualiza el empaquetado de plugins.
// Lo llama el build de KatPlugins autenticandose con X-Sync-Token.
func (s *Server) handleAdminSyncPlugins(w http.ResponseWriter, r *http.Request) {
	if !s.syncAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "sync no autorizado")
		return
	}

	var req syncRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	upserted := 0
	for _, p := range req.Plugins {
		if p.Slug == "" || p.JarFile == "" {
			continue
		}
		patch := map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"package":     p.Package,
			"jar_file":    p.JarFile,
			"version":     p.Version,
			"sha256":      p.Sha256,
		}

		existing, err := s.pluginBySlug(r.Context(), p.Slug)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "no se pudo consultar el plugin")
			return
		}

		if existing != nil {
			q := url.Values{}
			q.Set("slug", "eq."+p.Slug)
			if err := s.db.Patch(r.Context(), "plugins", q, patch); err != nil {
				writeError(w, http.StatusInternalServerError, "no se pudo actualizar "+p.Slug)
				return
			}
		} else {
			patch["slug"] = p.Slug
			if p.Name == "" {
				patch["name"] = p.Slug
			}
			tier := p.RequiredTier
			if !validTier(tier) {
				tier = "free"
			}
			patch["required_tier"] = tier
			if err := s.db.Insert(r.Context(), "plugins", nil, patch); err != nil {
				writeError(w, http.StatusInternalServerError, "no se pudo crear "+p.Slug)
				return
			}
		}
		upserted++
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "upserted": upserted})
}

func (s *Server) syncAuthorized(r *http.Request) bool {
	if s.cfg.SyncToken != "" {
		got := r.Header.Get("X-Sync-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.SyncToken)) == 1 {
			return true
		}
	}
	if st := s.currentUser(r); st != nil && st.IsAdmin {
		return true
	}
	return false
}
