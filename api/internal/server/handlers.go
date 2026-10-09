package server

import (
	"net/http"
	"net/url"

	"github.com/PaJauKat/katplugins-web/api/internal/models"
)

// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"supabaseConfigured": s.db.Configured(),
	})
}

// GET /api/plugins — catalogo publico. Si hay sesion, marca el acceso.
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.listPlugins(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar el catálogo de plugins")
		return
	}

	st := s.tryAuth(r)
	tier := "anon"
	overrides := map[string]bool{}
	if st != nil {
		tier = s.reconcileTier(r.Context(), st.Profile)
		if o, err := s.loadOverrides(r.Context(), st.Profile.ID); err == nil {
			overrides = o
		}
	}

	access := s.buildAccess(effectiveTierForAnon(tier), plugins, overrides)
	writeJSON(w, http.StatusOK, map[string]any{"plugins": access})
}

// GET /api/me — perfil + acceso a plugins del usuario autenticado.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	st := stateFrom(r.Context())

	plugins, err := s.listPlugins(r.Context(), st.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar el catálogo de plugins")
		return
	}
	overrides, err := s.loadOverrides(r.Context(), st.Profile.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los accesos")
		return
	}

	st.Profile.Tier = s.reconcileTier(r.Context(), st.Profile)

	writeJSON(w, http.StatusOK, models.MeResponse{
		User:    *st.Profile,
		IsAdmin: st.IsAdmin,
		Plugins: s.buildAccess(st.Profile.Tier, plugins, overrides),
	})
}

// GET /api/admin/users
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := url.Values{}
	q.Set("select", "*")
	q.Set("order", "created_at.desc")
	var profiles []dbProfile
	if err := s.db.Select(r.Context(), "profiles", q, &profiles); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo listar los usuarios")
		return
	}

	qAcc := url.Values{}
	qAcc.Set("select", "user_id,plugin_id,granted")
	var access []dbAccess
	if err := s.db.Select(r.Context(), "plugin_access", qAcc, &access); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los accesos")
		return
	}

	granted := map[string]int{}
	revoked := map[string]int{}
	for _, a := range access {
		if a.Granted {
			granted[a.UserID]++
		} else {
			revoked[a.UserID]++
		}
	}

	out := make([]models.AdminUser, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, models.AdminUser{
			User:         p.toModel(),
			GrantedCount: granted[p.ID],
			RevokedCount: revoked[p.ID],
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

type setTierRequest struct {
	UserID string `json:"userId"`
	Tier   string `json:"tier"`
}

// POST /api/admin/users/tier
func (s *Server) handleAdminSetTier(w http.ResponseWriter, r *http.Request) {
	var req setTierRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if req.UserID == "" || !validTier(req.Tier) {
		writeError(w, http.StatusBadRequest, "userId o tier inválido")
		return
	}

	q := url.Values{}
	q.Set("id", "eq."+req.UserID)
	if err := s.db.Patch(r.Context(), "profiles", q, map[string]any{"tier": req.Tier}); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el tier")
		return
	}

	st := stateFrom(r.Context())
	s.audit(r.Context(), st.Profile.ID, "set_tier", req.UserID, "", map[string]any{"tier": req.Tier})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/admin/users/access?userId=...
func (s *Server) handleAdminUserAccess(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "falta userId")
		return
	}

	q := url.Values{}
	q.Set("id", "eq."+userID)
	q.Set("select", "*")
	var profiles []dbProfile
	if err := s.db.Select(r.Context(), "profiles", q, &profiles); err != nil || len(profiles) == 0 {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	plugins, err := s.listPlugins(r.Context(), true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los plugins")
		return
	}
	overrides, err := s.loadOverrides(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los accesos")
		return
	}

	writeJSON(w, http.StatusOK, models.MeResponse{
		User:    profiles[0].toModel(),
		Plugins: s.buildAccess(profiles[0].Tier, plugins, overrides),
	})
}

type setAccessRequest struct {
	UserID   string `json:"userId"`
	PluginID string `json:"pluginId"`
	Granted  *bool  `json:"granted"`
}

// POST /api/admin/access — granted=null borra el override (vuelve al tier).
func (s *Server) handleAdminSetAccess(w http.ResponseWriter, r *http.Request) {
	var req setAccessRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if req.UserID == "" || req.PluginID == "" {
		writeError(w, http.StatusBadRequest, "userId y pluginId son obligatorios")
		return
	}

	st := stateFrom(r.Context())

	if req.Granted == nil {
		q := url.Values{}
		q.Set("user_id", "eq."+req.UserID)
		q.Set("plugin_id", "eq."+req.PluginID)
		if err := s.db.Delete(r.Context(), "plugin_access", q); err != nil {
			writeError(w, http.StatusInternalServerError, "no se pudo quitar el acceso")
			return
		}
		s.audit(r.Context(), st.Profile.ID, "clear_access", req.UserID, req.PluginID, nil)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "override": nil})
		return
	}

	payload := map[string]any{
		"user_id":   req.UserID,
		"plugin_id": req.PluginID,
		"granted":   *req.Granted,
	}
	if err := s.db.Upsert(r.Context(), "plugin_access", nil, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el acceso")
		return
	}
	s.audit(r.Context(), st.Profile.ID, "set_access", req.UserID, req.PluginID, map[string]any{"granted": *req.Granted})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "override": *req.Granted})
}

// GET /api/admin/plugins
func (s *Server) handleAdminListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.listPlugins(r.Context(), true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar los plugins")
		return
	}
	out := make([]models.Plugin, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p.toModel(s.cfg.PluginsBaseURL))
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": out})
}

type upsertPluginRequest struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RequiredTier string `json:"requiredTier"`
	Active       *bool  `json:"active"`
	SortOrder    *int   `json:"sortOrder"`
}

// POST /api/admin/plugins — crea o actualiza un plugin.
func (s *Server) handleAdminUpsertPlugin(w http.ResponseWriter, r *http.Request) {
	var req upsertPluginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if req.Slug == "" || req.Name == "" || !validTier(req.RequiredTier) {
		writeError(w, http.StatusBadRequest, "slug, name y requiredTier son obligatorios")
		return
	}

	payload := map[string]any{
		"slug":          req.Slug,
		"name":          req.Name,
		"description":   req.Description,
		"required_tier": req.RequiredTier,
	}
	if req.Active != nil {
		payload["active"] = *req.Active
	}
	if req.SortOrder != nil {
		payload["sort_order"] = *req.SortOrder
	}

	if req.ID != "" {
		payload["id"] = req.ID
	}

	if err := s.db.Upsert(r.Context(), "plugins", nil, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo guardar el plugin")
		return
	}

	st := stateFrom(r.Context())
	s.audit(r.Context(), st.Profile.ID, "upsert_plugin", "", req.ID, map[string]any{"slug": req.Slug})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// DELETE /api/admin/plugins?id=...
func (s *Server) handleAdminDeletePlugin(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "falta id")
		return
	}
	q := url.Values{}
	q.Set("id", "eq."+id)
	if err := s.db.Delete(r.Context(), "plugins", q); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo eliminar el plugin")
		return
	}
	st := stateFrom(r.Context())
	s.audit(r.Context(), st.Profile.ID, "delete_plugin", "", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// effectiveTierForAnon mapea un tier desconocido al minimo (free).
func effectiveTierForAnon(tier string) string {
	if validTier(tier) {
		return tier
	}
	return "free"
}
