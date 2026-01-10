package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Python ML service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new ML service client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithTimeout returns a new client with the specified timeout.
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	return &Client{
		baseURL: c.baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetHealth checks if the ML service is healthy.
func (c *Client) GetHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStatus retrieves the full status from the ML service.
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// GetMemoryUsage retrieves memory usage from the ML service.
func (c *Client) GetMemoryUsage(ctx context.Context) (*MemoryStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status/memory", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("memory request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var stats MemoryStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// IsHealthy performs a quick health check and returns true if the service is up.
func (c *Client) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.GetHealth(ctx) == nil
}

// Part 2 methods (interface ready, implementation for when Python service supports them)

// CheckVAD performs voice activity detection on an audio file.
// Part 2: Returns 501 Not Implemented until VAD is enabled.
func (c *Client) CheckVAD(ctx context.Context, audioPath string) (*VADResult, error) {
	reqBody := VADRequest{AudioPath: audioPath}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/vad/check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("VAD not implemented")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("VAD request failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result VADResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AskLLM sends a question to the LLM.
// Part 2: Returns 501 Not Implemented until LLM is enabled.
func (c *Client) AskLLM(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/llm/ask", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("LLM not implemented")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM request failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
