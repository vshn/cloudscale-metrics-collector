package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/cloudscale-metrics-collector/pkg/categoriesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/datetimesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/queriesmodel"
	"github.com/vshn/cloudscale-metrics-collector/pkg/tenantsmodel"
	"net/http"
	"os"
	"strconv"
	"time"
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

	// we must use the correct zones, otherwise the appuio-odoo-adapter will not work correctly
	sourceZones = []string{"c-appuio-cloudscale-lpg-2"}

	// source "

	// products
	productsData = []*db.Product{
		{
			Source: sourceQueryStorage + ":" + sourceZones[0],
			Target: sql.NullString{String: "1401", Valid: true},
			Amount: 0.003,
			Unit:   "GBDay", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryTrafficOut + ":" + sourceZones[0],
			Target: sql.NullString{String: "1403", Valid: true},
			Amount: 0.02,
			Unit:   "GB", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryRequests + ":" + sourceZones[0],
			Target: sql.NullString{String: "1405", Valid: true},
			Amount: 0.005,
			Unit:   "KReq",
			During: db.InfiniteRange(),
		},
	}

	discountsData = []*db.Discount{
		{
			Source:   sourceQueryStorage,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
		{
			Source:   sourceQueryTrafficOut,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
		{
			Source:   sourceQueryRequests,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
	}

	queriesData = []*db.Query{
		{
			Name:        sourceQueryStorage + ":" + sourceZones[0],
			Description: "Object Storage - Storage (cloudscale.ch)",
			Query:       "",
			Unit:        "GBDay",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryTrafficOut + ":" + sourceZones[0],
			Description: "Object Storage - Traffic Out (cloudscale.ch)",
			Query:       "",
			Unit:        "GB",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryRequests + ":" + sourceZones[0],
			Description: "Object Storage - Requests (cloudscale.ch)",
			Query:       "",
			Unit:        "KReq",
			During:      db.InfiniteRange(),
		},
	}
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
	for _, product := range productsData {
		_, err := productsmodel.Ensure(ctx, tx, product)
		if err != nil {
			return err
		}
	}

	for _, discount := range discountsData {
		_, err := discountsmodel.Ensure(ctx, tx, discount)
		if err != nil {
			return err
		}
	}

	for _, query := range queriesData {
		_, err := queriesmodel.Ensure(ctx, tx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	ctx := context.Background()
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
	defer tx.Rollback()
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
