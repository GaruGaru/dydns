package providers

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

type Porkbun struct {
	client    *http.Client
	apiKey    string
	apiSecret string
	records   []string
	domainID  string
}

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
	for _, record := range p.records {
		err := p.updateRecord(ctx, record, addr)
		if err != nil {
			return fmt.Errorf("failed to update dns record: %w", err)
		}
	}
	return nil
}

func (p *Porkbun) updateRecord(ctx context.Context, record string, addr string) error {
	pbReq := porkbunRequestEdit{
		Secretapikey: p.apiSecret,
		Apikey:       p.apiKey,
		Name:         record,
		Type:         "A",
		Content:      addr,
		Ttl:          "600",
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(pbReq)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	endpoint, err := url.JoinPath("https://api.porkbun.com/api/json/v3", "/dns/editByNameType/", p.domainID, "A", record)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
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

type porkbunRequestEdit struct {
	Secretapikey string `json:"secretapikey"`
	Apikey       string `json:"apikey"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	Ttl          string `json:"ttl"`
}
