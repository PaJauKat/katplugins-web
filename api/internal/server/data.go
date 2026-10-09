package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"

	"github.com/PaJauKat/katplugins-web/api/internal/auth"
	"github.com/PaJauKat/katplugins-web/api/internal/models"
)

type dbProfile struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	GoogleSub      string `json:"google_sub"`
	FullName       string `json:"full_name"`
	AvatarURL      string `json:"avatar_url"`
	Role           string `json:"role"`
	Tier           string `json:"tier"`
	FlowCustomerID string `json:"flow_customer_id"`
	CreatedAt      string `json:"created_at"`
}

type dbPlugin struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RequiredTier string `json:"required_tier"`
	Active       bool   `json:"active"`
	SortOrder    int    `json:"sort_order"`
	Package      string `json:"package"`
	JarFile      string `json:"jar_file"`
	Version      string `json:"version"`
	Sha256       string `json:"sha256"`
}

type dbAccess struct {
	UserID   string `json:"user_id"`
	PluginID string `json:"plugin_id"`
	Granted  bool   `json:"granted"`
}

func (p dbProfile) toModel() models.User {
	return models.User{
		ID:        p.ID,
		Email:     p.Email,
		FullName:  p.FullName,
		AvatarURL: p.AvatarURL,
		Role:      p.Role,
		Tier:      p.Tier,
		CreatedAt: p.CreatedAt,
	}
}

func (p dbPlugin) toModel(baseURL string) models.Plugin {
	m := models.Plugin{
		ID:           p.ID,
		Slug:         p.Slug,
		Name:         p.Name,
		Description:  p.Description,
		RequiredTier: p.RequiredTier,
		Active:       p.Active,
		SortOrder:    p.SortOrder,
		Package:      p.Package,
		JarFile:      p.JarFile,
		Version:      p.Version,
		Sha256:       p.Sha256,
	}
	if p.JarFile != "" && baseURL != "" {
		m.URL = baseURL + "/plugins/" + p.JarFile
	}
	return m
}

var validTiers = map[string]int{"free": 0, "plus": 1, "pro": 2}

func tierRank(t string) int {
	if r, ok := validTiers[t]; ok {
		return r
	}
	return 0
}

func validTier(t string) bool {
	_, ok := validTiers[t]
	return ok
}

// getProfile devuelve el perfil por id, o nil si no existe.
func (s *Server) getProfile(ctx context.Context, id string) (*models.User, error) {
	q := url.Values{}
	q.Set("id", "eq."+id)
	q.Set("select", "*")
	q.Set("limit", "1")

	var rows []dbProfile
	if err := s.db.Select(ctx, "profiles", q, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	m := rows[0].toModel()
	return &m, nil
}

// findOrCreateGoogleUser busca (por google_sub o email) o crea el perfil.
func (s *Server) findOrCreateGoogleUser(ctx context.Context, gu *auth.GoogleUser) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(gu.Email))
	if email == "" {
		return nil, fmt.Errorf("google no devolvio email")
	}
	normalized := *gu
	normalized.Email = email
	gu = &normalized

	if existing, err := s.profileByColumn(ctx, "google_sub", gu.Sub); err != nil {
		return nil, err
	} else if existing != nil {
		return s.syncGoogleProfile(ctx, existing, gu)
	}

	if existing, err := s.profileByColumn(ctx, "email", gu.Email); err != nil {
		return nil, err
	} else if existing != nil {
		return s.syncGoogleProfile(ctx, existing, gu)
	}

	id := newUUID()
	payload := map[string]any{
		"id":         id,
		"email":      gu.Email,
		"google_sub": gu.Sub,
		"full_name":  gu.Name,
		"avatar_url": gu.Picture,
	}
	if err := s.db.Insert(ctx, "profiles", nil, payload); err != nil {
		return nil, err
	}

	return &models.User{
		ID:        id,
		Email:     gu.Email,
		FullName:  gu.Name,
		AvatarURL: gu.Picture,
		Role:      "user",
		Tier:      "free",
	}, nil
}

func (s *Server) profileByColumn(ctx context.Context, column, value string) (*dbProfile, error) {
	if value == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set(column, "eq."+value)
	q.Set("select", "*")
	q.Set("limit", "1")

	var rows []dbProfile
	if err := s.db.Select(ctx, "profiles", q, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *Server) syncGoogleProfile(ctx context.Context, existing *dbProfile, gu *auth.GoogleUser) (*models.User, error) {
	patch := map[string]any{}
	if existing.GoogleSub != gu.Sub {
		patch["google_sub"] = gu.Sub
	}
	if gu.Name != "" && existing.FullName != gu.Name {
		patch["full_name"] = gu.Name
	}
	if gu.Picture != "" && existing.AvatarURL != gu.Picture {
		patch["avatar_url"] = gu.Picture
	}
	if gu.Email != "" && existing.Email != gu.Email {
		patch["email"] = gu.Email
	}

	if len(patch) > 0 {
		q := url.Values{}
		q.Set("id", "eq."+existing.ID)
		if err := s.db.Patch(ctx, "profiles", q, patch); err != nil {
			return nil, err
		}
		if v, ok := patch["google_sub"].(string); ok {
			existing.GoogleSub = v
		}
		if v, ok := patch["full_name"].(string); ok {
			existing.FullName = v
		}
		if v, ok := patch["avatar_url"].(string); ok {
			existing.AvatarURL = v
		}
		if v, ok := patch["email"].(string); ok {
			existing.Email = v
		}
	}

	m := existing.toModel()
	return &m, nil
}

// pluginBySlug devuelve un plugin por slug, o nil si no existe.
func (s *Server) pluginBySlug(ctx context.Context, slug string) (*dbPlugin, error) {
	if slug == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("slug", "eq."+slug)
	q.Set("select", "*")
	q.Set("limit", "1")

	var rows []dbPlugin
	if err := s.db.Select(ctx, "plugins", q, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// listPlugins devuelve el catalogo. Si includeInactive es false, solo activos.
func (s *Server) listPlugins(ctx context.Context, includeInactive bool) ([]dbPlugin, error) {
	q := url.Values{}
	q.Set("select", "*")
	q.Set("order", "sort_order.asc,name.asc")
	if !includeInactive {
		q.Set("active", "is.true")
	}
	var rows []dbPlugin
	if err := s.db.Select(ctx, "plugins", q, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// loadOverrides devuelve el mapa pluginID -> granted para un usuario.
func (s *Server) loadOverrides(ctx context.Context, userID string) (map[string]bool, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("select", "plugin_id,granted")

	var rows []dbAccess
	if err := s.db.Select(ctx, "plugin_access", q, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.PluginID] = r.Granted
	}
	return out, nil
}

// buildAccess calcula el acceso efectivo de un tier + overrides.
func (s *Server) buildAccess(userTier string, plugins []dbPlugin, overrides map[string]bool) []models.PluginAccess {
	rank := tierRank(userTier)
	result := make([]models.PluginAccess, 0, len(plugins))
	for _, p := range plugins {
		access := rank >= tierRank(p.RequiredTier)
		source := "tier"
		var override *bool
		if granted, ok := overrides[p.ID]; ok {
			access = granted
			g := granted
			override = &g
			if granted {
				source = "granted"
			} else {
				source = "revoked"
			}
		}
		result = append(result, models.PluginAccess{
			Plugin:   p.toModel(s.cfg.PluginsBaseURL),
			Access:   access,
			Source:   source,
			Override: override,
		})
	}
	return result
}

// audit registra una accion de administracion (best-effort, no rompe la request).
func (s *Server) audit(ctx context.Context, actorID, action, targetUserID, pluginID string, detail map[string]any) {
	if detail == nil {
		detail = map[string]any{}
	}
	row := map[string]any{
		"actor_id":       nullableID(actorID),
		"action":         action,
		"target_user_id": nullableID(targetUserID),
		"plugin_id":      nullableID(pluginID),
		"detail":         detail,
	}
	_ = s.db.Insert(ctx, "audit_log", nil, row)
}

func nullableID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// newUUID genera un UUID v4 sin dependencias externas.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
