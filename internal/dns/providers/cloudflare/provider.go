package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go_cmdb/internal/dnstypes"
)

const (
	cloudflareAPIBase = "https://api.cloudflare.com/client/v4"
	requestTimeout    = 10 * time.Second
)

var (
	// ErrNotFound is returned when a DNS record is not found
	ErrNotFound = errors.New("DNS record not found")
)

// CloudflareProvider implements dns.Provider for Cloudflare API
type CloudflareProvider struct {
	email    string
	apiToken string
	client   *http.Client
}

// NewCloudflareProvider creates a new Cloudflare DNS provider
func NewCloudflareProvider(email, apiToken string) *CloudflareProvider {
	return &CloudflareProvider{
		email:    email,
		apiToken: apiToken,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// CloudflareRecord represents a Cloudflare DNS record (API response)
type CloudflareRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// CloudflareResponse represents a Cloudflare API response
type CloudflareResponse struct {
	Success bool                `json:"success"`
	Errors  []CloudflareError   `json:"errors"`
	Result  json.RawMessage     `json:"result"`
}

// CloudflareError represents a Cloudflare API error
type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// EnsureRecord ensures a DNS record exists with the correct values
func (p *CloudflareProvider) EnsureRecord(zoneID string, record dnstypes.DNSRecord) (string, bool, error) {
	// Step 1: Find existing record
	existingID, err := p.FindRecord(zoneID, record.Type, record.Name, record.Value)
	if err != nil && err != ErrNotFound {
		return "", false, fmt.Errorf("failed to find existing record: %w", err)
	}

	// Step 2: If record exists, check if update is needed
	if existingID != "" {
		// Record exists, check if values match
		existing, err := p.getRecord(zoneID, existingID)
		if err != nil {
			return existingID, false, fmt.Errorf("failed to get existing record: %w", err)
		}

		// Check if update is needed
		if existing.TTL == record.TTL && existing.Proxied == record.Proxied {
			// No change needed
			return existingID, false, nil
		}

		// Update record
		if err := p.updateRecord(zoneID, existingID, record); err != nil {
			return existingID, false, fmt.Errorf("failed to update record: %w", err)
		}

		return existingID, true, nil
	}

	// Step 3: Create new record
	recordID, err := p.createRecord(zoneID, record)
	if err != nil {
		return "", false, fmt.Errorf("failed to create record: %w", err)
	}

	return recordID, true, nil
}

	// DeleteRecord deletes a DNS record by its provider-specific ID
// Returns ErrNotFound if the record doesn't exist (treated as success for deletion)
func (p *CloudflareProvider) DeleteRecord(zoneID string, providerRecordID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareAPIBase, zoneID, providerRecordID)

	req, err := http.NewRequestWithContext(context.Background(), "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for 404 - record not found (treat as success)
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		// Check for record not found error code (81044)
		for _, e := range cfResp.Errors {
			if e.Code == 81044 || e.Code == 81043 {
				return ErrNotFound
			}
		}
		return fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	return nil
}

// FindRecord finds a DNS record by type, name, and value
func (p *CloudflareProvider) FindRecord(zoneID string, recordType string, name string, value string) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s&content=%s", 
		cloudflareAPIBase, zoneID, recordType, name, value)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return "", fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	var records []CloudflareRecord
	if err := json.Unmarshal(cfResp.Result, &records); err != nil {
		return "", fmt.Errorf("failed to parse result: %w", err)
	}

	if len(records) == 0 {
		return "", ErrNotFound
	}

	return records[0].ID, nil
}

// createRecord creates a new DNS record
func (p *CloudflareProvider) createRecord(zoneID string, record dnstypes.DNSRecord) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", cloudflareAPIBase, zoneID)

	payload := map[string]interface{}{
		"type":    record.Type,
		"name":    record.Name,
		"content": record.Value,
		"ttl":     record.TTL,
		"proxied": record.Proxied,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return "", fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	var createdRecord CloudflareRecord
	if err := json.Unmarshal(cfResp.Result, &createdRecord); err != nil {
		return "", fmt.Errorf("failed to parse result: %w", err)
	}

	return createdRecord.ID, nil
}

// updateRecord updates an existing DNS record
func (p *CloudflareProvider) updateRecord(zoneID string, recordID string, record dnstypes.DNSRecord) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareAPIBase, zoneID, recordID)

	payload := map[string]interface{}{
		"type":    record.Type,
		"name":    record.Name,
		"content": record.Value,
		"ttl":     record.TTL,
		"proxied": record.Proxied,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	return nil
}

// getRecord gets a DNS record by ID
func (p *CloudflareProvider) getRecord(zoneID string, recordID string) (*CloudflareRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareAPIBase, zoneID, recordID)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
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

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	var record CloudflareRecord
	if err := json.Unmarshal(cfResp.Result, &record); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &record, nil
}

// formatErrors formats Cloudflare API errors into a readable string
func formatErrors(errors []CloudflareError) string {
	if len(errors) == 0 {
		return "unknown error"
	}

	var errMsgs []string
	for _, e := range errors {
		errMsgs = append(errMsgs, fmt.Sprintf("[%d] %s", e.Code, e.Message))
	}

	return fmt.Sprintf("%v", errMsgs)
}

// ListRecords lists all DNS records for a zone
func (p *CloudflareProvider) ListRecords(ctx context.Context, zoneID string) ([]CloudflareRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?per_page=1000", cloudflareAPIBase, zoneID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Auth-Email", p.email)
	req.Header.Set("X-Auth-Key", p.apiToken)
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

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("cloudflare API error: %s", formatErrors(cfResp.Errors))
	}

	var records []CloudflareRecord
	if err := json.Unmarshal(cfResp.Result, &records); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return records, nil
}
