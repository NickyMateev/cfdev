package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"gopkg.in/segmentio/analytics-go.v3"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfdevos "code.cloudfoundry.org/cfdev/os"
	"code.cloudfoundry.org/cfdev/pkg/analyticsd/daemon"
	"github.com/denisbrodbeck/machineid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	analyticsKey     string
	testAnalyticsKey string
	version          string
	pollingInterval  = 10 * time.Minute
)

func main() {
	cfg := &clientcredentials.Config{
		ClientID:     "analytics",
		ClientSecret: "analytics",
		TokenURL:     "https://uaa.dev.cfdev.sh/oauth/token",
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	userID, err := machineid.ProtectedID("cfdev")
	if err != nil {
		userID = "UNKNOWN_ID"
	}

	o := cfdevos.OS{}
	osVersion, err := o.Version()
	if err != nil {
		osVersion = "unknown-os-version"
	}

	if os.Getenv("CFDEV_MODE") == "debug" {
		pollingInterval = 10 * time.Second
	}

	var analytixKey string
	if os.Getenv("CFDEV_MODE") == "debug" || analyticsKey == "" {
		analytixKey = testAnalyticsKey
	} else {
		analytixKey = analyticsKey
	}

	analyticsDaemon := daemon.New(
		"https://api.dev.cfdev.sh",
		userID,
		version,
		osVersion,
		os.Stdout,
		cfg.Client(ctx),
		analytics.New(analytixKey),
		pollingInterval,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		analyticsDaemon.Stop()
	}()

	fmt.Printf("[ANALYTICSD] apiKeyLoaded: %t, pollingInterval: %v, version: %q, time: %v, userID: %q\n",
		analyticsKey != "", pollingInterval, version, time.Now(), userID)
	analyticsDaemon.Start()
}
