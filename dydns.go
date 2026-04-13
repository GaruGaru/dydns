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

	var options = providers.Options{
		Domain:   os.Getenv("DOMAIN"),
		Entries:  strings.Split(os.Getenv("ENTRIES"), ","),
		Password: os.Getenv("PASSWORD"),
	}

	if len(options.Password) == 0 {
		logger.Error("password is required")
		os.Exit(1)
	}

	if len(options.Entries) == 0 {
		logger.Error("atleast 1 entry is required")
		os.Exit(1)
	}

	if len(options.Domain) == 0 {
		logger.Error("domain is required")
		os.Exit(1)
	}

	delay := 60 * time.Second

	delayEnv := os.Getenv("DELAY")
	var err error
	if len(delayEnv) != 0 {
		delay, err = time.ParseDuration(delayEnv)
		if err != nil {
			logger.Error("failed to parse delay", "error", err)
			os.Exit(1)
		}
	}

	dnsClient := providers.NewDnsClient()
	logger.Info("Starting dydns", "domain", options.Domain, "entries", len(options.Entries))

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
				err = dnsClient.Update(ctx, options, externalIP)
				if err != nil {
					logger.Warn("error updating DNS record", "error", err)
					continue
				}
				logger.Info("updated DNS records", "count", len(options.Entries))
			}
		}
	}()

	<-done
	logger.Info("dydns exiting")
}
