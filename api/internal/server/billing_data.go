package server

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/PaJauKat/katplugins-web/api/internal/models"
)

type dbSubscription struct {
	ID               string  `json:"id"`
	UserID           string  `json:"user_id"`
	Provider         string  `json:"provider"`
	Plan             string  `json:"plan"`
	Interval         string  `json:"interval"`
	Status           string  `json:"status"`
	ExternalID       string  `json:"external_id"`
	CheckoutRef      string  `json:"checkout_ref"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	CurrentPeriodEnd string  `json:"current_period_end"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func (s *Server) listSubscriptions(ctx context.Context, userID string) ([]dbSubscription, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("select", "*")
	q.Set("order", "created_at.desc")

	var rows []dbSubscription
	if err := s.db.Select(ctx, "subscriptions", q, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Server) subscriptionByID(ctx context.Context, id string) (*dbSubscription, error) {
	if id == "" {
		return nil, nil
	}
	return s.firstSubscription(ctx, url.Values{"id": {"eq." + id}})
}

func (s *Server) subscriptionByExternal(ctx context.Context, provider, externalID string) (*dbSubscription, error) {
	if externalID == "" {
		return nil, nil
	}
	return s.firstSubscription(ctx, url.Values{
		"provider":    {"eq." + provider},
		"external_id": {"eq." + externalID},
	})
}

func (s *Server) firstSubscription(ctx context.Context, q url.Values) (*dbSubscription, error) {
	q.Set("select", "*")
	q.Set("limit", "1")
	var rows []dbSubscription
	if err := s.db.Select(ctx, "subscriptions", q, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *Server) insertSubscription(ctx context.Context, sub dbSubscription) error {
	payload := map[string]any{
		"id":       sub.ID,
		"user_id":  sub.UserID,
		"provider": sub.Provider,
		"plan":     sub.Plan,
		"interval": sub.Interval,
		"status":   sub.Status,
		"amount":   sub.Amount,
		"currency": sub.Currency,
	}
	if sub.ExternalID != "" {
		payload["external_id"] = sub.ExternalID
	}
	if sub.CheckoutRef != "" {
		payload["checkout_ref"] = sub.CheckoutRef
	}
	if sub.CurrentPeriodEnd != "" {
		payload["current_period_end"] = sub.CurrentPeriodEnd
	}
	return s.db.Insert(ctx, "subscriptions", nil, payload)
}

func (s *Server) patchSubscription(ctx context.Context, id string, patch map[string]any) error {
	q := url.Values{}
	q.Set("id", "eq."+id)
	return s.db.Patch(ctx, "subscriptions", q, patch)
}

func (s *Server) setFlowCustomerID(ctx context.Context, userID, customerID string) error {
	q := url.Values{}
	q.Set("id", "eq."+userID)
	return s.db.Patch(ctx, "profiles", q, map[string]any{"flow_customer_id": customerID})
}

func (s *Server) setUserTier(ctx context.Context, userID, tier string) error {
	q := url.Values{}
	q.Set("id", "eq."+userID)
	return s.db.Patch(ctx, "profiles", q, map[string]any{"tier": tier})
}

// subscriptionActive indica si una suscripcion otorga acceso ahora mismo.
func subscriptionActive(sub dbSubscription, now time.Time) bool {
	if sub.Status != "active" && sub.Status != "past_due" {
		return false
	}
	if strings.TrimSpace(sub.CurrentPeriodEnd) == "" {
		return true
	}
	if t, ok := parseDBTime(sub.CurrentPeriodEnd); ok {
		return t.After(now)
	}
	return true
}

// reconcileTier ajusta el tier del perfil segun sus suscripciones.
// No toca cuentas admin ni perfiles sin ninguna suscripcion registrada
// (para no pisar accesos otorgados manualmente).
func (s *Server) reconcileTier(ctx context.Context, profile *models.User) string {
	if profile == nil || profile.Role == "admin" || profile.Tier == "pro" {
		return profile.Tier
	}
	subs, err := s.listSubscriptions(ctx, profile.ID)
	if err != nil || len(subs) == 0 {
		return profile.Tier
	}

	now := time.Now()
	active := false
	hasSettled := false
	for _, sub := range subs {
		if sub.Status == "pending" {
			continue
		}
		hasSettled = true
		if subscriptionActive(sub, now) {
			active = true
			break
		}
	}
	if !hasSettled {
		return profile.Tier
	}

	desired := "free"
	if active {
		desired = "plus"
	}
	if desired == profile.Tier {
		return profile.Tier
	}
	if err := s.setUserTier(ctx, profile.ID, desired); err != nil {
		return profile.Tier
	}
	return desired
}

// parseDBTime interpreta timestamps de Postgres/Supabase en sus formatos usuales.
func parseDBTime(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
