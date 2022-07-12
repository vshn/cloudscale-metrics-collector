package main

import (
	"fmt"
	"net/http"
	"os"
	//	"os/signal"
	//	"sync/atomic"
	//	"syscall"
	"context"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"time"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")

	appName     = "cloudscale-metrics-collector"
	appLongName = "cloudscale-metrics-collector"

	// constants
	tokenEnvVariable = "CLOUDSCALE_API_TOKEN"
)

func main() {
	ctx := context.Background()
	cloudscaleApiToken := os.Getenv(tokenEnvVariable)
	if cloudscaleApiToken == "" {
		fmt.Fprintf(os.Stderr, "Environment variable %s must be set", tokenEnvVariable)
		os.Exit(1)
	}

	cloudscaleClient := cloudscale.NewClient(http.DefaultClient)
	cloudscaleClient.AuthToken = cloudscaleApiToken

	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startTime := endTime.AddDate(0, -6, 0)
	fmt.Printf("start: %s, end: %s\n", startTime, endTime)
	bucketMetricsRequest := cloudscale.BucketMetricsRequest{Start: startTime, End: endTime}
	var bucketMetrics *cloudscale.BucketMetrics
	var err error
	bucketMetrics, err = cloudscaleClient.Metrics.GetBucketMetrics(ctx, &bucketMetricsRequest)
	if err != nil {
		fmt.Printf("ERROR: " + err.Error() + "\n")
		os.Exit(1)
	}

	fmt.Printf("result: %+v\n", bucketMetrics.Data)
}
