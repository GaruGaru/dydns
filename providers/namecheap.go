package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Options struct {
	Domain   string
	Entries  []string
	Password string
}

func NewNamecheap() *NamecheapClient {
	return &NamecheapClient{
		client: &http.Client{Timeout: time.Second * 15},
	}
}

type NamecheapClient struct {
	client *http.Client
}

func (c *NamecheapClient) Update(ctx context.Context, options Options, ip string) error {
	for _, entry := range options.Entries {
		url := fmt.Sprintf(
			"https://dynamicdns.park-your-domain.com/update?domain=%s&host=%s&password=%s&ip=%s",
			options.Domain,
			entry,
			options.Password,
			ip,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := c.client.Do(req)

		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	return nil
}
