package main

import (
	"fmt"
	"os"
	//	"os/signal"
	//	"sync/atomic"
	//	"syscall"
	"time"
	//	"github.com/go-logr/logr"
	//	"github.com/urfave/cli/v2"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")

	appName     = "cloudscale-metrics-collector"
	appLongName = "cloudscale-metrics-collector"

	// TODO: Adjust or clear env var prefix
	// envPrefix is the global prefix to use for the keys in environment variables
	envPrefix = "CLOUDSCALE_METRICS_COLLECTOR"
)

func env(suffix string) string {
	return os.Getenv(envPrefix + "_" + suffix)
}

func main() {
	fmt.Printf("hello world\n")
	fmt.Printf(env("PATH") + "\n")
}
