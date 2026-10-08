package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client es un cliente minimo para la API REST de Supabase (PostgREST).
// Usa la secret key, que bypassa RLS, por lo que solo debe invocarse desde
// el backend tras validar la sesion del usuario.
type Client struct {
	baseURL   string
	secretKey string
	http      *http.Client
}

func New(baseURL, secretKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		secretKey: secretKey,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// Configured indica si el cliente tiene los datos minimos para operar.
func (c *Client) Configured() bool {
	return c.baseURL != "" && c.secretKey != ""
}

func (c *Client) restURL(path string, q url.Values) string {
	base := fmt.Sprintf("%s/rest/v1/%s", c.baseURL, path)
	if len(q) > 0 {
		return base + "?" + q.Encode()
	}
	return base
}

// Select ejecuta una consulta GET sobre una tabla/vista y decodifica el JSON.
func (c *Client) Select(ctx context.Context, path string, q url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.restURL(path, q), nil)
	if err != nil {
		return err
	}
	body, status, err := c.doAdmin(req)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("supabase GET %s %d: %s", path, status, string(body))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// Upsert inserta o actualiza filas (resolution=merge-duplicates).
func (c *Client) Upsert(ctx context.Context, path string, q url.Values, rows interface{}) error {
	return c.write(ctx, http.MethodPost, path, q, rows, "resolution=merge-duplicates,return=representation")
}

// Insert inserta filas sin fusionar.
func (c *Client) Insert(ctx context.Context, path string, q url.Values, rows interface{}) error {
	return c.write(ctx, http.MethodPost, path, q, rows, "return=representation")
}

// Patch actualiza filas que cumplan el filtro.
func (c *Client) Patch(ctx context.Context, path string, q url.Values, patch interface{}) error {
	return c.write(ctx, http.MethodPatch, path, q, patch, "return=representation")
}

// Delete elimina filas que cumplan el filtro.
func (c *Client) Delete(ctx context.Context, path string, q url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.restURL(path, q), nil)
	if err != nil {
		return err
	}
	body, status, err := c.doAdmin(req)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("supabase DELETE %s %d: %s", path, status, string(body))
	}
	return nil
}

func (c *Client) write(ctx context.Context, method, path string, q url.Values, payload interface{}, prefer string) error {
	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.restURL(path, q), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", prefer)

	body, status, err := c.doAdmin(req)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("supabase %s %s %d: %s", method, path, status, string(body))
	}
	return nil
}

func (c *Client) doAdmin(req *http.Request) ([]byte, int, error) {
	req.Header.Set("apikey", c.secretKey)
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
