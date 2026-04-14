package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Porkbun struct {
	client    *http.Client
	apiKey    string
	apiSecret string
	records   []string
	domainID  string
}

const (
	baseURL = "https://api.porkbun.com/api/json/v3"
)

func NewPorkbun(apiKey string, apiSecret string, domainID string, records []string) *Porkbun {
	return &Porkbun{
		client:    &http.Client{Timeout: 15 * time.Second},
		apiKey:    apiKey,
		apiSecret: apiSecret,
		domainID:  domainID,
		records:   records,
	}
}

func (p *Porkbun) Update(ctx context.Context, addr string) error {
	logger := slog.Default()
	for _, record := range p.records {
		curr, err := p.currentRecord(ctx, record)
		if err != nil {
			return fmt.Errorf("failed to get current record: %w", err)
		}

		if curr == addr {
			logger.Info("record already updated")
			continue
		}

		err = p.updateRecord(ctx, record, addr)
		if err != nil {
			return fmt.Errorf("failed to update dns record: %w", err)
		}
	}
	return nil
}

func (p *Porkbun) updateRecord(ctx context.Context, record string, addr string) error {
	pbReq := porkbunRequestEdit{
		porkbunAuthRequest: porkbunAuthRequest{
			Secretapikey: p.apiSecret,
			Apikey:       p.apiKey,
		},
		Name:    record,
		Type:    "A",
		Content: addr,
		Ttl:     "600",
	}

	encoded, err := json.Marshal(pbReq)
	if err != nil {
		return fmt.Errorf("failed to encode delete request: %w", err)
	}

	endpoint, err := url.JoinPath(baseURL, "/dns/editByNameType/", p.domainID, "A", record)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer req.Body.Close()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *Porkbun) currentRecord(ctx context.Context, record string) (string, error) {
	pReq := porkbunAuthRequest{
		Secretapikey: p.apiSecret,
		Apikey:       p.apiKey,
	}

	type response struct {
		Records []struct {
			Content string `json:"content"`
		} `json:"records"`
	}

	encoded, err := json.Marshal(pReq)
	if err != nil {
		return "", fmt.Errorf("failed to encode delete request: %w", err)
	}

	endpoint, err := url.JoinPath(baseURL, "/dns/retrieveByNameType/", p.domainID, "A", record)
	if err != nil {
		return "", fmt.Errorf("failed to delete endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("failed to delete endpoint: %w", err)
	}
	defer req.Body.Close()

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer req.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed with status code %d: %s %s", resp.StatusCode, endpoint, string(respBody))
	}

	var res response
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(res.Records) == 0 {
		return "", nil
	}

	if len(res.Records) > 1 {
		return "", fmt.Errorf("more than one record returned")
	}

	return res.Records[0].Content, nil

}

type porkbunAuthRequest struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
}

type porkbunRequestEdit struct {
	porkbunAuthRequest
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Ttl     string `json:"ttl"`
}
