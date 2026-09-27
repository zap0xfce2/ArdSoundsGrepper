// Package api spricht die GraphQL-API der ARD Audiothek (ARD Sounds) an.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultEndpoint = "https://api.ardaudiothek.de/graphql"
	endpointEnv     = "ARDSOUNDS_API"
	requestTimeout  = 30 * time.Second
	UserAgent       = "ardsoundsgrepper"
)

var ErrShowNotFound = errors.New("Sendung nicht gefunden")

type Client struct {
	endpoint string
	http     *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{endpoint: endpoint, http: &http.Client{Timeout: requestTimeout}}
}

// DefaultEndpoint erlaubt Tests und Docker, die API per ARDSOUNDS_API umzubiegen.
func DefaultEndpoint() string {
	if endpoint := os.Getenv(endpointEnv); endpoint != "" {
		return endpoint
	}
	return defaultEndpoint
}

type graphQLRequest[V any] struct {
	Query     string `json:"query"`
	Variables V      `json:"variables"`
}

type graphQLResponse[D any] struct {
	Data   D `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func post[V, D any](ctx context.Context, c *Client, query string, variables V, data *D) error {
	body, err := json.Marshal(graphQLRequest[V]{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("API nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()
	return decodeResponse(resp, data)
}

func decodeResponse[D any](resp *http.Response, data *D) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API antwortet mit HTTP %d", resp.StatusCode)
	}
	var envelope graphQLResponse[D]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("API-Antwort unlesbar: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("GraphQL-Fehler: %s", envelope.Errors[0].Message)
	}
	*data = envelope.Data
	return nil
}
