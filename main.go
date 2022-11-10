package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/vshn/cloudscale-metrics-collector/pkg/categoriesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/datetimesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/queriesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/tenantsmodel"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/jmoiron/sqlx"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")
	appName = "cloudscale-metrics-collector"

	// constants
	daysEnvVariable  = "DAYS"
	tokenEnvVariable = "CLOUDSCALE_API_TOKEN"
	dbUrlEnvVariable = "ACR_DB_URL"

	// source format: 'query:zone:tenant:namespace' or 'query:zone:tenant:namespace:class'
	// We do not have real (prometheus) queries here, just random hardcoded strings.
	sourceQueryStorage    = "object-storage-storage"
	sourceQueryTrafficOut = "object-storage-traffic-out"
	sourceQueryRequests   = "object-storage-requests"

	// SourceZone represents the zone of the bucket, not of the cluster where the request for the bucket originated.
	// All the zones we use here must be known to the appuio-odoo-adapter as well.
	sourceZones = []string{"cloudscale"}
)

func cfg() (string, string, int) {
	cloudscaleApiToken := os.Getenv(tokenEnvVariable)
	if cloudscaleApiToken == "" {
		fmt.Fprintf(os.Stderr, "ERROR: Environment variable %s must be set\n", tokenEnvVariable)
		os.Exit(1)
	}

	dbUrl := os.Getenv(dbUrlEnvVariable)
	if dbUrl == "" {
		fmt.Fprintf(os.Stderr, "ERROR: Environment variable %s must be set\n", dbUrlEnvVariable)
		os.Exit(1)
	}

	daysStr := os.Getenv(daysEnvVariable)
	if daysStr == "" {
		daysStr = "1"
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Environment variable %s must contain an integer\n", daysEnvVariable)
		os.Exit(1)
	}

	return cloudscaleApiToken, dbUrl, days
}

func initDb(ctx context.Context, tx *sqlx.Tx) error {
	for _, product := range ensureProducts {
		_, err := productsmodel.Ensure(ctx, tx, product)
		if err != nil {
			return err
		}
	}

	for _, discount := range ensureDiscounts {
		_, err := discountsmodel.Ensure(ctx, tx, discount)
		if err != nil {
			return err
		}
	}

	for _, query := range ensureQueries {
		_, err := queriesmodel.Ensure(ctx, tx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "%s: version %s (%s) compiled on %s\n", appName, version, commit, date)

	err := sync(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func sync(ctx context.Context) error {
	cloudscaleApiToken, dbUrl, days := cfg()

	cloudscaleClient := cloudscale.NewClient(http.DefaultClient)
	cloudscaleClient.AuthToken = cloudscaleApiToken

	// The cloudscale API works in Europe/Zurich, so we have to use the same, regardless of where this code runs
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return err
	}

	// Fetch statistics of yesterday (as per Europe/Zurich). The metrics will cover the entire day.
	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-days, 0, 0, 0, 0, now.Location())
	if err != nil {
		return err
	}

	rdb, err := db.Openx(dbUrl)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// initialize DB
	tx, err := rdb.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Fprintf(os.Stderr, "rollback failed: %v", err)
		}
	}(tx)
	err = initDb(ctx, tx)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	accumulated, err := accumulateBucketMetrics(ctx, date, cloudscaleClient)
	if err != nil {
		return err
	}

	for source, value := range accumulated {
		if value == 0 {
			continue
		}

		fmt.Printf("syncing %s\n", source)

		// start new transaction for actual work
		tx, err = rdb.BeginTxx(ctx, &sql.TxOptions{})
		if err != nil {
			return err
		}

		tenant, err := tenantsmodel.Ensure(ctx, tx, &db.Tenant{Source: source.Tenant})
		if err != nil {
			return err
		}

		category, err := categoriesmodel.Ensure(ctx, tx, &db.Category{Source: source.Zone + ":" + source.Namespace})
		if err != nil {
			return err
		}

		dateTime := datetimesmodel.New(source.Start)
		dateTime, err = datetimesmodel.Ensure(ctx, tx, dateTime)
		if err != nil {
			return err
		}

		product, err := productsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		if err != nil {
			return err
		}

		discount, err := discountsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		if err != nil {
			return err
		}

		query, err := queriesmodel.GetByName(ctx, tx, source.Query+":"+source.Zone)
		if err != nil {
			return err
		}

		var quantity float64
		if query.Unit == "GB" || query.Unit == "GBDay" {
			quantity = float64(value) / 1000 / 1000 / 1000
		} else if query.Unit == "KReq" {
			quantity = float64(value) / 1000
		} else {
			return errors.New("Unknown query unit " + query.Unit)
		}
		storageFact := factsmodel.New(dateTime, query, tenant, category, product, discount, quantity)
		_, err = factsmodel.Ensure(ctx, tx, storageFact)
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}
