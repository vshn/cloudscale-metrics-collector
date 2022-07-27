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
	dbUrlEnvVariable = "ACR_DB_URL"

	// source format: 'query:zone:tenant:namespace' or 'query:zone:tenant:namespace:class'
	// We do not have real (prometheus) queries here, just random hardcoded strings.
	sourceQueryStorage    = "s3-storage"
	sourceQueryTrafficOut = "s3-traffic-out"
	sourceQueryRequests   = "s3-requests"

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
			Description: "S3 Storage",
			Query:       "",
			Unit:        "GBDay",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryTrafficOut + ":" + sourceZones[0],
			Description: "S3 Traffic Out",
			Query:       "",
			Unit:        "GB",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryRequests + ":" + sourceZones[0],
			Description: "S3 Requests",
			Query:       "",
			Unit:        "KReq",
			During:      db.InfiniteRange(),
		},
	}
)

func cfg() (string, string) {
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

	return cloudscaleApiToken, dbUrl
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

func checkErrExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()
	cloudscaleApiToken, dbUrl := cfg()

	cloudscaleClient := cloudscale.NewClient(http.DefaultClient)
	cloudscaleClient.AuthToken = cloudscaleApiToken

	// The cloudscale API works in Europe/Zurich, so we have to use the same, regardless of where this code runs
	location, err := time.LoadLocation("Europe/Zurich")
	checkErrExit(err)

	// Fetch statistics of yesterday (as per Europe/Zurich). The metrics will cover the entire day.
	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	checkErrExit(err)

	rdb, err := db.Openx(dbUrl)
	checkErrExit(err)
	defer rdb.Close()

	// initialize DB
	tx, err := rdb.BeginTxx(ctx, &sql.TxOptions{})
	checkErrExit(err)
	defer tx.Rollback()
	err = initDb(ctx, tx)
	checkErrExit(err)
	err = tx.Commit()
	checkErrExit(err)

	accumulated, err := accumulate(ctx, date, cloudscaleClient)
	checkErrExit(err)

	for source, value := range accumulated {
		if value == 0 {
			continue
		}

		fmt.Printf("syncing %s\n", source)

		// start new transaction for actual work
		tx, err = rdb.BeginTxx(ctx, &sql.TxOptions{})
		checkErrExit(err)

		tenant, err := tenantsmodel.Ensure(ctx, tx, &db.Tenant{Source: source.Tenant})
		checkErrExit(err)

		category, err := categoriesmodel.Ensure(ctx, tx, &db.Category{Source: source.Zone + ":" + source.Namespace})
		checkErrExit(err)

		dateTime := datetimesmodel.New(source.Start)
		dateTime, err = datetimesmodel.Ensure(ctx, tx, dateTime)
		checkErrExit(err)

		product, err := productsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		checkErrExit(err)

		discount, err := discountsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		checkErrExit(err)

		query, err := queriesmodel.GetByName(ctx, tx, source.Query+":"+source.Zone)
		checkErrExit(err)

		var quantity float64
		if query.Unit == "GB" || query.Unit == "GBDay" {
			quantity = float64(value) / 1000 / 1000 / 1000
		} else if query.Unit == "KReq" {
			quantity = float64(value) / 1000
		} else {
			checkErrExit(errors.New("Unknown query unit " + query.Unit))
		}
		storageFact := factsmodel.New(dateTime, query, tenant, category, product, discount, quantity)
		_, err = factsmodel.Ensure(ctx, tx, storageFact)
		checkErrExit(err)

		err = tx.Commit()
		checkErrExit(err)
	}

	os.Exit(0)
}
