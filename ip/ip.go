package ip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func Providers(providers ...Provider) Provider {
	return MultiProvider{
		Providers: providers,
	}
}

type MultiProvider struct {
	Providers []Provider
}

func (p MultiProvider) IP(ctx context.Context) (string, error) {
	for _, provider := range p.Providers {
		ip, err := provider.IP(ctx)
		if err == nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("no working provider")
}

func (p MultiProvider) Name() string {
	return "providers-manager"
}

type Provider interface {
	IP(context.Context) (string, error)
	Name() string
}

func NewPlainIPProvider(url string) Provider {
	return PlainIPProvider{
		client: http.Client{
			Timeout: time.Second * 15,
		},
		url: url,
	}
}

type PlainIPProvider struct {
	url    string
	client http.Client
}

func (p PlainIPProvider) IP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (p PlainIPProvider) Name() string {
	return p.url
}
