package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PaJauKat/katplugins-web/api/internal/models"
	"github.com/PaJauKat/katplugins-web/api/internal/payments"
)

const (
	plusMonthlyCLP = 2000
	plusAnnualCLP  = 20000
	plusMonthlyUSD = 3
	plusAnnualUSD  = 30
)

// GET /api/billing/providers — configuracion publica de planes y pasarelas.
func (s *Server) handleBillingProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"plan": map[string]any{
			"id":   "plus",
			"name": "Plus",
			"prices": map[string]any{
				"monthly": map[string]int{"clp": plusMonthlyCLP, "usd": plusMonthlyUSD},
				"annual":  map[string]int{"clp": plusAnnualCLP, "usd": plusAnnualUSD},
			},
		},
		"providers": []map[string]any{
			{
				"id":        "flow",
				"name":      "Flow (CLP)",
				"currency":  "CLP",
				"intervals": []string{"monthly", "annual"},
				"enabled":   s.cfg.FlowEnabled(),
			},
			{
				"id":        "lemonsqueezy",
				"name":      "Tarjeta",
				"currency":  s.cfg.LemonCurrency,
				"intervals": []string{"monthly", "annual"},
				"enabled":   s.cfg.LemonEnabled(),
			},
			{
				"id":        "nowpayments",
				"name":      "Cripto (anual)",
				"currency":  "USD",
				"intervals": []string{"annual"},
				"enabled":   s.cfg.NowPaymentsEnabled(),
			},
		},
	})
}

type checkoutRequest struct {
	Provider string `json:"provider"`
	Interval string `json:"interval"`
}

// POST /api/billing/checkout — crea una sesion de pago y devuelve la URL.
func (s *Server) handleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	st := stateFrom(r.Context())

	var req checkoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	interval := strings.ToLower(strings.TrimSpace(req.Interval))
	if interval != "monthly" && interval != "annual" {
		writeError(w, http.StatusBadRequest, "intervalo inválido")
		return
	}

	ctx := r.Context()
	var redirectURL string
	var err error

	switch provider {
	case "flow":
		redirectURL, err = s.startFlowCheckout(ctx, st.Profile, interval)
	case "lemonsqueezy":
		redirectURL, err = s.startLemonCheckout(ctx, st.Profile, interval)
	case "nowpayments":
		if interval != "annual" {
			writeError(w, http.StatusBadRequest, "cripto solo admite suscripción anual")
			return
		}
		redirectURL, err = s.startNowPaymentsCheckout(ctx, st.Profile, interval)
	default:
		writeError(w, http.StatusBadRequest, "pasarela no soportada")
		return
	}

	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": redirectURL})
}

func (s *Server) newPendingSubscription(ctx context.Context, userID, provider, interval string, amount int, currency string) (dbSubscription, error) {
	sub := dbSubscription{
		ID:          newUUID(),
		UserID:      userID,
		Provider:    provider,
		Plan:        "plus",
		Interval:    interval,
		Status:      "pending",
		CheckoutRef: newUUID(),
		Amount:      float64(amount),
		Currency:    currency,
	}
	if err := s.insertSubscription(ctx, sub); err != nil {
		return sub, err
	}
	return sub, nil
}

// startFlowCheckout crea/reutiliza el cliente Flow y redirige al registro de tarjeta.
func (s *Server) startFlowCheckout(ctx context.Context, profile *models.User, interval string) (string, error) {
	if !s.cfg.FlowEnabled() {
		return "", errNotConfigured("Flow")
	}
	row, err := s.profileByColumn(ctx, "id", profile.ID)
	if err != nil || row == nil {
		return "", errNotConfigured("perfil")
	}

	customerID := row.FlowCustomerID
	flow := payments.NewFlow(s.cfg.FlowBaseURL, s.cfg.FlowAPIKey, s.cfg.FlowSecretKey)
	if customerID == "" {
		name := profile.FullName
		if strings.TrimSpace(name) == "" {
			name = profile.Email
		}
		customerID, err = flow.CreateCustomer(ctx, name, profile.Email, profile.ID)
		if err != nil {
			return "", err
		}
		if err := s.setFlowCustomerID(ctx, profile.ID, customerID); err != nil {
			return "", err
		}
	}

	amount := plusMonthlyCLP
	if interval == "annual" {
		amount = plusAnnualCLP
	}
	sub, err := s.newPendingSubscription(ctx, profile.ID, "flow", interval, amount, "CLP")
	if err != nil {
		return "", err
	}

	returnURL := s.cfg.AppBaseURL + "/api/billing/flow/return?sub=" + sub.ID
	redirect, token, err := flow.RegisterCard(ctx, customerID, returnURL)
	if err != nil {
		return "", err
	}
	if err := s.patchSubscription(ctx, sub.ID, map[string]any{"checkout_ref": token}); err != nil {
		return "", err
	}
	return redirect + "?token=" + token, nil
}

// startLemonCheckout crea un checkout hospedado de LemonSqueezy.
func (s *Server) startLemonCheckout(ctx context.Context, profile *models.User, interval string) (string, error) {
	if !s.cfg.LemonEnabled() {
		return "", errNotConfigured("LemonSqueezy")
	}
	variantID := s.cfg.LemonVariantID(interval)
	amount := plusMonthlyUSD
	if interval == "annual" {
		amount = plusAnnualUSD
	}
	if _, err := s.newPendingSubscription(ctx, profile.ID, "lemonsqueezy", interval, amount, "USD"); err != nil {
		return "", err
	}
	lemon := payments.NewLemonSqueezy(s.cfg.LemonAPIKey)
	return lemon.CreateCheckout(
		ctx,
		s.cfg.LemonStoreID,
		variantID,
		s.cfg.AppBaseURL+"/dashboard?billing=ok",
		profile.Email,
		profile.FullName,
		profile.ID,
	)
}

// startNowPaymentsCheckout crea una factura cripto (solo anual).
func (s *Server) startNowPaymentsCheckout(ctx context.Context, profile *models.User, interval string) (string, error) {
	if !s.cfg.NowPaymentsEnabled() {
		return "", errNotConfigured("NowPayments")
	}
	sub, err := s.newPendingSubscription(ctx, profile.ID, "nowpayments", interval, plusAnnualUSD, "USD")
	if err != nil {
		return "", err
	}
	np := payments.NewNowPayments(s.cfg.NowPaymentsAPIKey)
	url, _, err := np.CreateInvoice(
		ctx,
		s.cfg.NowPaymentsAnnualUSD,
		"usd",
		sub.ID,
		"KatPlugins Plus (anual)",
		s.cfg.AppBaseURL+"/api/webhooks/nowpayments",
		s.cfg.AppBaseURL+"/dashboard?billing=ok",
		s.cfg.AppBaseURL+"/#precios",
	)
	return url, err
}

// GET /api/billing/subscription — estado actual + reconciliacion de tier.
func (s *Server) handleBillingSubscription(w http.ResponseWriter, r *http.Request) {
	st := stateFrom(r.Context())
	ctx := r.Context()

	subs, err := s.listSubscriptions(ctx, st.Profile.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo cargar la suscripción")
		return
	}

	// Reconciliar suscripciones Flow (renovaciones / cancelaciones).
	for i := range subs {
		sub := subs[i]
		if sub.Provider == "flow" && sub.ExternalID != "" && sub.Status == "active" {
			if updated, err := s.refreshFlowSubscription(ctx, sub); err == nil && updated != nil {
				subs[i] = *updated
			}
		}
	}

	st.Profile.Tier = s.reconcileTier(ctx, st.Profile)

	writeJSON(w, http.StatusOK, map[string]any{
		"tier":          st.Profile.Tier,
		"subscriptions": subs,
	})
}

// refreshFlowSubscription consulta Flow y actualiza la fila local.
func (s *Server) refreshFlowSubscription(ctx context.Context, sub dbSubscription) (*dbSubscription, error) {
	flow := payments.NewFlow(s.cfg.FlowBaseURL, s.cfg.FlowAPIKey, s.cfg.FlowSecretKey)
	res, err := flow.GetSubscription(ctx, sub.ExternalID)
	if err != nil {
		return nil, err
	}
	patch := flowSubscriptionPatch(res)
	if len(patch) == 0 {
		return nil, nil
	}
	if err := s.patchSubscription(ctx, sub.ID, patch); err != nil {
		return nil, err
	}
	if v, ok := patch["status"].(string); ok {
		sub.Status = v
	}
	if v, ok := patch["current_period_end"].(string); ok {
		sub.CurrentPeriodEnd = v
	}
	return &sub, nil
}

func flowSubscriptionPatch(res map[string]any) map[string]any {
	patch := map[string]any{}
	if status, ok := res["status"]; ok {
		patch["status"] = payments.FlowStatusText(status)
	}
	if end, ok := payments.FlowTime(res["period_end"]); ok {
		patch["current_period_end"] = end.UTC().Format(time.RFC3339)
	} else if end, ok := payments.FlowTime(res["subscription_end"]); ok {
		patch["current_period_end"] = end.UTC().Format(time.RFC3339)
	}
	return patch
}

// handleFlowReturn maneja el retorno de Flow tras registrar la tarjeta.
// Acepta GET (redireccion del navegador) y POST (notificacion de Flow).
func (s *Server) handleFlowReturn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subID := r.URL.Query().Get("sub")
	if subID == "" {
		http.Redirect(w, r, "/dashboard?billing=error", http.StatusFound)
		return
	}
	sub, err := s.subscriptionByID(ctx, subID)
	if err != nil || sub == nil {
		http.Redirect(w, r, "/dashboard?billing=error", http.StatusFound)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		token = sub.CheckoutRef
	}

	flow := payments.NewFlow(s.cfg.FlowBaseURL, s.cfg.FlowAPIKey, s.cfg.FlowSecretKey)
	res, err := flow.GetRegisterStatus(ctx, token)
	ok := err == nil && strings.TrimSpace(toString(res["status"])) == "1"
	customerID := toString(res["customerId"])
	if customerID == "" {
		if row, _ := s.profileByColumn(ctx, "id", sub.UserID); row != nil {
			customerID = row.FlowCustomerID
		}
	}

	if ok && customerID != "" {
		planID := s.cfg.FlowPlanID(sub.Interval)
		subRes, err := flow.CreateSubscription(ctx, planID, customerID)
		if err == nil {
			patch := flowSubscriptionPatch(subRes)
			if id := toString(subRes["subscriptionId"]); id != "" {
				patch["external_id"] = id
			}
			if patch["status"] == nil {
				patch["status"] = "active"
			}
			_ = s.patchSubscription(ctx, sub.ID, patch)
			if profile, _ := s.getProfile(ctx, sub.UserID); profile != nil {
				s.reconcileTier(ctx, profile)
			}
		}
	}

	if r.Method == http.MethodPost {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	http.Redirect(w, r, "/dashboard?billing=ok", http.StatusFound)
}

// POST /api/webhooks/flow — notificacion del plan (cobros recurrentes).
func (s *Server) handleFlowWebhook(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	// Confirmar que el pago existe (best-effort); el estado se reconcilia al abrir el panel.
	flow := payments.NewFlow(s.cfg.FlowBaseURL, s.cfg.FlowAPIKey, s.cfg.FlowSecretKey)
	_, _ = flow.GetPaymentStatus(r.Context(), token)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type lemonWebhook struct {
	Meta struct {
		EventName  string `json:"event_name"`
		CustomData struct {
			UserID string `json:"user_id"`
		} `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Status    string `json:"status"`
			RenewsAt  string `json:"renews_at"`
			EndsAt    string `json:"ends_at"`
			VariantID any    `json:"variant_id"`
		} `json:"attributes"`
	} `json:"data"`
}

// POST /api/webhooks/lemonsqueezy
func (s *Server) handleLemonWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	sig := r.Header.Get("X-Signature")
	if !payments.VerifyLemonWebhook(s.cfg.LemonWebhookSecret, body, sig) {
		writeError(w, http.StatusUnauthorized, "firma inválida")
		return
	}

	var payload lemonWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	ctx := r.Context()

	userID := payload.Meta.CustomData.UserID
	externalID := payload.Data.ID
	status := mapLemonStatus(payload.Data.Attributes.Status)
	periodEnd := payload.Data.Attributes.RenewsAt
	if strings.TrimSpace(periodEnd) == "" {
		periodEnd = payload.Data.Attributes.EndsAt
	}

	sub, _ := s.subscriptionByExternal(ctx, "lemonsqueezy", externalID)
	if sub == nil && userID != "" {
		sub = s.latestPendingSubscription(ctx, userID, "lemonsqueezy")
	}
	if sub == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	patch := map[string]any{"status": status, "external_id": externalID}
	if t, ok := parseDBTime(periodEnd); ok {
		patch["current_period_end"] = t.UTC().Format(time.RFC3339)
	}
	_ = s.patchSubscription(ctx, sub.ID, patch)

	if profile, _ := s.getProfile(ctx, sub.UserID); profile != nil {
		s.reconcileTier(ctx, profile)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func mapLemonStatus(status string) string {
	switch strings.ToLower(status) {
	case "active", "on_trial":
		return "active"
	case "past_due", "unpaid":
		return "past_due"
	case "paused":
		return "cancelled"
	case "cancelled", "expired":
		return "expired"
	default:
		return "pending"
	}
}

func (s *Server) latestPendingSubscription(ctx context.Context, userID, provider string) *dbSubscription {
	subs, err := s.listSubscriptions(ctx, userID)
	if err != nil {
		return nil
	}
	for i := range subs {
		if subs[i].Provider == provider && subs[i].Status == "pending" {
			return &subs[i]
		}
	}
	return nil
}

// POST /api/webhooks/nowpayments
func (s *Server) handleNowPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	sig := r.Header.Get("x-nowpayments-sig")
	if !payments.VerifyNowPaymentsIPN(s.cfg.NowPaymentsIPNSecret, body, sig) {
		writeError(w, http.StatusUnauthorized, "firma inválida")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "json inválido")
		return
	}
	ctx := r.Context()

	subID := toString(payload["order_id"])
	status := strings.ToLower(toString(payload["payment_status"]))
	sub, err := s.subscriptionByID(ctx, subID)
	if err != nil || sub == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	switch status {
	case "finished", "confirmed":
		end := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)
		_ = s.patchSubscription(ctx, sub.ID, map[string]any{
			"status":             "active",
			"current_period_end": end,
		})
		if profile, _ := s.getProfile(ctx, sub.UserID); profile != nil {
			s.reconcileTier(ctx, profile)
		}
	case "failed", "expired", "refunded":
		_ = s.patchSubscription(ctx, sub.ID, map[string]any{"status": "expired"})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strings.TrimRight(strings.TrimRight(formatFloat(t), "0"), ".")
	default:
		return ""
	}
}

func formatFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

type notConfiguredError string

func (e notConfiguredError) Error() string { return string(e) + " no está configurado" }

func errNotConfigured(name string) error { return notConfiguredError(name) }
