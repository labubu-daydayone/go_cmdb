package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Zone represents a Cloudflare Zone (domain)
type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	NameServers []string `json:"name_servers"`
}

// ListZonesResponse represents the Cloudflare API response for listing zones
type ListZonesResponse struct {
	Success bool   `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result []Zone `json:"result"`
}

// ListZones retrieves all zones from Cloudflare account
func (p *CloudflareProvider) ListZones(ctx context.Context) ([]Zone, error) {
	url := fmt.Sprintf("%s/zones", cloudflareAPIBase)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp ListZonesResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare API returned success=false")
	}

	return apiResp.Result, nil
}
