package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxErrorBody = 1 << 20

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("api url must be an absolute http or https url")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: baseURL, http: httpClient}, nil
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.request(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, payload, result any) error {
	return c.request(ctx, http.MethodPost, path, payload, result)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.request(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) request(ctx context.Context, method, path string, payload, result any) error {
	if c == nil || c.http == nil {
		return errors.New("api client is not initialized")
	}
	if !strings.HasPrefix(path, "/") {
		return errors.New("api path must start with a slash")
	}

	var body io.Reader
	if payload != nil {
		encoded := new(bytes.Buffer)
		if err := json.NewEncoder(encoded).Encode(payload); err != nil {
			return fmt.Errorf("encode api request: %w", err)
		}
		body = encoded
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build api request: %w", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("connect to rewindfs: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
		var apiError struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &apiError) == nil && apiError.Error != "" {
			return fmt.Errorf("rewindfs: %s", apiError.Error)
		}
		return fmt.Errorf("rewindfs returned %s", response.Status)
	}

	if result == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode rewindfs response: %w", err)
	}
	return nil
}
