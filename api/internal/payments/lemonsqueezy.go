package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LemonSqueezy es un cliente minimo de la API de LemonSqueezy.
type LemonSqueezy struct {
	apiKey string
	http   *http.Client
}

func NewLemonSqueezy(apiKey string) *LemonSqueezy {
	return &LemonSqueezy{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

type lemonCheckoutRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			ProductOptions map[string]any `json:"product_options"`
			CheckoutData   map[string]any `json:"checkout_data"`
			CheckoutOpts   map[string]any `json:"checkout_options,omitempty"`
		} `json:"attributes"`
		Relationships struct {
			Store struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"store"`
			Variant struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"variant"`
		} `json:"relationships"`
	} `json:"data"`
}

// CreateCheckout crea un checkout hospedado y devuelve su URL publica.
func (l *LemonSqueezy) CreateCheckout(ctx context.Context, storeID, variantID, redirectURL, email, name, userID string) (string, error) {
	var body lemonCheckoutRequest
	body.Data.Type = "checkouts"
	body.Data.Attributes.ProductOptions = map[string]any{
		"redirect_url":     redirectURL,
		"enabled_variants": []string{variantID},
	}
	custom := map[string]any{"user_id": userID}
	checkoutData := map[string]any{"custom": custom}
	if email != "" {
		checkoutData["email"] = email
	}
	if name != "" {
		checkoutData["name"] = name
	}
	body.Data.Attributes.CheckoutData = checkoutData
	body.Data.Attributes.CheckoutOpts = map[string]any{"button_color": "#dd1f2f"}
	body.Data.Relationships.Store.Data.Type = "stores"
	body.Data.Relationships.Store.Data.ID = storeID
	body.Data.Relationships.Variant.Data.Type = "variants"
	body.Data.Relationships.Variant.Data.ID = variantID

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.lemonsqueezy.com/v1/checkouts", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("lemonsqueezy checkout %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", err
	}
	if out.Data.Attributes.URL == "" {
		return "", fmt.Errorf("lemonsqueezy: checkout sin url")
	}
	return out.Data.Attributes.URL, nil
}

// VerifyLemonWebhook valida la firma X-Signature (HMAC-SHA256 hex del cuerpo).
func VerifyLemonWebhook(secret string, rawBody []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
