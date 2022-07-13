package main

import (
	"context"
	"fmt"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"net/http"
	"os"
	"time"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")
	appName = "cloudscale-metrics-collector"

	// constants
	tokenEnvVariable = "CLOUDSCALE_API_TOKEN"
)

func main() {
	ctx := context.Background()
	cloudscaleApiToken := os.Getenv(tokenEnvVariable)
	if cloudscaleApiToken == "" {
		fmt.Fprintf(os.Stderr, "ERROR: Environment variable %s must be set", tokenEnvVariable)
		os.Exit(1)
	}

	cloudscaleClient := cloudscale.NewClient(http.DefaultClient)
	cloudscaleClient.AuthToken = cloudscaleApiToken

	// The cloudscale API works in Europe/Zurich, so we have to use the same, regardless of where this code runs
	var err error
	var location *time.Location
	location, err = time.LoadLocation("Europe/Zurich")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: "+err.Error()+"\n")
		os.Exit(1)
	}

	// Fetch statistics of yesterday (as per Europe/Zurich). The metrics will cover the entire day.
	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	bucketMetricsRequest := cloudscale.BucketMetricsRequest{Start: date, End: date}
	var bucketMetrics *cloudscale.BucketMetrics
	bucketMetrics, err = cloudscaleClient.Metrics.GetBucketMetrics(ctx, &bucketMetricsRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: "+err.Error()+"\n")
		os.Exit(1)
	}

	fmt.Printf("%+v\n", bucketMetrics.Data)
}
