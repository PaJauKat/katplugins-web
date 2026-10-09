package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NowPayments es un cliente minimo de la API de NOWPayments (cripto).
type NowPayments struct {
	apiKey string
	http   *http.Client
}

func NewNowPayments(apiKey string) *NowPayments {
	return &NowPayments{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

type nowPaymentsInvoice struct {
	PriceAmount    string `json:"price_amount"`
	PriceCurrency  string `json:"price_currency"`
	OrderID        string `json:"order_id"`
	OrderDesc      string `json:"order_description,omitempty"`
	IPNCallbackURL string `json:"ipn_callback_url,omitempty"`
	SuccessURL     string `json:"success_url,omitempty"`
	CancelURL      string `json:"cancel_url,omitempty"`
}

// CreateInvoice crea una factura de pago en cripto y devuelve su URL.
func (n *NowPayments) CreateInvoice(ctx context.Context, priceAmount, priceCurrency, orderID, description, ipnURL, successURL, cancelURL string) (string, string, error) {
	payload := nowPaymentsInvoice{
		PriceAmount:    priceAmount,
		PriceCurrency:  priceCurrency,
		OrderID:        orderID,
		OrderDesc:      description,
		IPNCallbackURL: ipnURL,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.nowpayments.io/v1/invoice", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("x-api-key", n.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("nowpayments invoice %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		InvoiceURL string `json:"invoice_url"`
		ID         any    `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", err
	}
	if out.InvoiceURL == "" {
		return "", "", fmt.Errorf("nowpayments: invoice sin url")
	}
	return out.InvoiceURL, fmt.Sprintf("%v", out.ID), nil
}

// VerifyNowPaymentsIPN valida el header x-nowpayments-sig: HMAC-SHA512 (hex)
// sobre el JSON con las claves ordenadas alfabeticamente.
func VerifyNowPaymentsIPN(secret string, rawBody []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(rawBody))
	dec.UseNumber()
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return false
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return false
	}
	canonical := bytes.TrimRight(buf.Bytes(), "\n")

	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(canonical)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}
