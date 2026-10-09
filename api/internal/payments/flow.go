// Package payments agrupa los clientes de las pasarelas de pago soportadas:
// Flow (CLP, suscripciones), LemonSqueezy (USD, suscripciones) y NowPayments
// (cripto, pago anual).
package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Flow es un cliente minimo de la API REST de Flow (flow.cl).
type Flow struct {
	baseURL   string
	apiKey    string
	secretKey string
	http      *http.Client
}

func NewFlow(baseURL, apiKey, secretKey string) *Flow {
	return &Flow{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		secretKey: secretKey,
		http:      &http.Client{Timeout: 20 * time.Second},
	}
}

// sign firma los parametros segun el esquema de Flow: ordenar por nombre,
// concatenar nombre+valor y aplicar HMAC-SHA256 con la secretKey (hex).
func (f *Flow) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "s" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(params[k])
	}

	mac := hmac.New(sha256.New, []byte(f.secretKey))
	mac.Write([]byte(sb.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

func (f *Flow) do(ctx context.Context, method, path string, params map[string]string) (map[string]any, error) {
	if params == nil {
		params = map[string]string{}
	}
	params["apiKey"] = f.apiKey
	params["s"] = f.sign(params)

	endpoint := f.baseURL + path

	var req *http.Request
	var err error
	if method == http.MethodGet {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	} else {
		form := url.Values{}
		for k, v := range params {
			form.Set(k, v)
		}
		req, err = http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return nil, err
	}

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	out := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &out)
	}
	if resp.StatusCode >= 400 {
		msg, _ := out["message"].(string)
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return out, fmt.Errorf("flow %s %d: %s", path, resp.StatusCode, msg)
	}
	return out, nil
}

// CreateCustomer registra un cliente en Flow y devuelve su customerId.
func (f *Flow) CreateCustomer(ctx context.Context, name, email, externalID string) (string, error) {
	res, err := f.do(ctx, http.MethodPost, "/customer/create", map[string]string{
		"name":       name,
		"email":      email,
		"externalId": externalID,
	})
	if err != nil {
		return "", err
	}
	id, _ := res["customerId"].(string)
	if id == "" {
		return "", fmt.Errorf("flow: customer/create no devolvio customerId")
	}
	return id, nil
}

// RegisterCard inicia el registro de tarjeta y devuelve la URL de redireccion.
func (f *Flow) RegisterCard(ctx context.Context, customerID, urlReturn string) (string, string, error) {
	res, err := f.do(ctx, http.MethodPost, "/customer/register", map[string]string{
		"customerId": customerID,
		"url_return": urlReturn,
	})
	if err != nil {
		return "", "", err
	}
	redirect, _ := res["url"].(string)
	token, _ := res["token"].(string)
	if redirect == "" || token == "" {
		return "", "", fmt.Errorf("flow: customer/register incompleto")
	}
	return redirect, token, nil
}

// GetRegisterStatus consulta el resultado del registro de tarjeta.
func (f *Flow) GetRegisterStatus(ctx context.Context, token string) (map[string]any, error) {
	return f.do(ctx, http.MethodGet, "/customer/getRegisterStatus", map[string]string{"token": token})
}

// CreateSubscription suscribe un cliente a un plan y devuelve la suscripcion.
func (f *Flow) CreateSubscription(ctx context.Context, planID, customerID string) (map[string]any, error) {
	return f.do(ctx, http.MethodPost, "/subscription/create", map[string]string{
		"planId":     planID,
		"customerId": customerID,
	})
}

// GetSubscription obtiene el estado de una suscripcion por su id.
func (f *Flow) GetSubscription(ctx context.Context, subscriptionID string) (map[string]any, error) {
	return f.do(ctx, http.MethodGet, "/subscription/get", map[string]string{"subscriptionId": subscriptionID})
}

// GetPaymentStatus obtiene el estado de un pago por token (webhooks).
func (f *Flow) GetPaymentStatus(ctx context.Context, token string) (map[string]any, error) {
	return f.do(ctx, http.MethodGet, "/payment/getStatus", map[string]string{"token": token})
}

// FlowTime parsea una fecha de Flow ("2006-01-02 15:04:05") a time.Time.
func FlowTime(v any) (time.Time, bool) {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(s), time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// FlowStatusText traduce el status numerico de una suscripcion.
func FlowStatusText(status any) string {
	n, ok := toInt(status)
	if !ok {
		return "pending"
	}
	switch n {
	case 1:
		return "active"
	case 2:
		return "active" // periodo de trial
	case 4:
		return "cancelled"
	default:
		return "pending"
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		return i, err == nil
	default:
		return 0, false
	}
}
