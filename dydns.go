package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/garugaru/DyDns/ip"
	"github.com/garugaru/DyDns/providers"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	var ipProvider = ip.Providers(
		ip.NewPlainIPProvider("https://api.ipify.org/"),
		ip.NewPlainIPProvider("http://myexternalip.com/raw"),
	)

	porkApiKey := os.Getenv("PORK_API_KEY")
	porkApiSecret := os.Getenv("PORK_API_SECRET")
	porkDomainID := os.Getenv("PORK_API_DOMAIN_ID")
	records := os.Getenv("RECORDS")

	if len(porkApiKey) == 0 {
		logger.Error("PORK_API_KEY is required")
		os.Exit(1)
	}

	if len(porkApiSecret) == 0 {
		logger.Error("PORK_API_SECRET is required")
		os.Exit(1)
	}

	if len(porkDomainID) == 0 {
		logger.Error("PORK_API_DOMAIN_ID is required")
		os.Exit(1)
	}

	if len(records) == 0 {
		logger.Error("RECORDS is required")
		os.Exit(1)
	}

	delay := 5 * time.Minute

	delayEnv := os.Getenv("DELAY")
	var err error
	if len(delayEnv) != 0 {
		delay, err = time.ParseDuration(delayEnv)
		if err != nil {
			logger.Error("failed to parse delay", "error", err)
			os.Exit(1)
		}
	}

	dnsClient := providers.NewPorkbun(porkApiKey, porkApiSecret, porkDomainID, strings.Split(records, ","))
	logger.Info("started dydns")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		logger.Warn("received signal", "signal", sig.String())
		cancel()
		done <- true
	}()

	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	go func() {
		var lastIP string
		for {
			select {
			case <-ctx.Done():
				logger.Info("context done, exiting")
				return
			case <-ticker.C:
				externalIP, err := ipProvider.IP(ctx)
				if err != nil {
					logger.Warn("error retrieving external IP", "error", err)
					continue
				}

				logger.Info("got IP", "ip", externalIP)

				if externalIP == lastIP {
					// skip redundant updates
					continue
				}

				err = dnsClient.Update(ctx, externalIP)
				if err != nil {
					logger.Error("error updating DNS record", "error", err)
					continue
				}
				lastIP = externalIP
				logger.Info("updated DNS records")
			}
		}
	}()

	<-done
	logger.Info("dydns exiting")
}
