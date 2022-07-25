package main

import (
	"context"
	"database/sql"
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
			Unit:   "GB/day", // SI GB according to cloudscale
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
			Unit:   "1k Requests",
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

	queryData = []*db.Query{
		{
			Name:        "Dummy",
			Description: "Dummy query for facts without queries",
			Query:       "",
			Unit:        "",
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

	for _, query := range queryData {
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
	bucketMetricsRequest := cloudscale.BucketMetricsRequest{Start: date, End: date}
	bucketMetrics, err := cloudscaleClient.Metrics.GetBucketMetrics(ctx, &bucketMetricsRequest)
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

	for _, bucketMetricsData := range bucketMetrics.Data {
		// start new transaction for actual work
		tx, err = rdb.BeginTxx(ctx, &sql.TxOptions{})
		checkErrExit(err)

		objectsUser, err := cloudscaleClient.ObjectsUsers.Get(ctx, bucketMetricsData.Subject.ObjectsUserID)
		if err != nil || objectsUser == nil {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s, objects user %s not found\n", bucketMetricsData.Subject.BucketName, bucketMetricsData.Subject.ObjectsUserID)
			continue
		}

		tenantStr := objectsUser.Tags["tenant"]
		if tenantStr == "" {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s, no tenant information found on objectsUser\n", bucketMetricsData.Subject.BucketName)
			continue
		}
		namespace := objectsUser.Tags["namespace"]
		if namespace == "" {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s, no namespace information found on objectsUser\n", bucketMetricsData.Subject.BucketName)
			continue
		}
		zone := objectsUser.Tags["zone"]
		if zone == "" {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s, no zone information found on objectsUser\n", bucketMetricsData.Subject.BucketName)
			continue
		}

		sourceStorage := sourceQueryStorage + ":" + zone + ":" + tenantStr + ":" + namespace
		sourceTrafficOut := sourceQueryTrafficOut + ":" + zone + ":" + tenantStr + ":" + namespace
		sourceRequests := sourceQueryRequests + ":" + zone + ":" + tenantStr + ":" + namespace

		tenant, err := tenantsmodel.Ensure(ctx, tx, &db.Tenant{Source: tenantStr})
		checkErrExit(err)

		category, err := categoriesmodel.Ensure(ctx, tx, &db.Category{Source: zone + ":" + objectsUser.DisplayName})
		checkErrExit(err)

		// Ensure a suitable dateTime object
		dateTime := datetimesmodel.New(bucketMetricsData.TimeSeries[0].Start)
		dateTime, err = datetimesmodel.Ensure(ctx, tx, dateTime)
		checkErrExit(err)

		// Find the right query. Since we don't actually query prometheus we just fetch a dummy object.
		query, err := queriesmodel.GetByName(ctx, tx, "Dummy")
		checkErrExit(err)

		if bucketMetricsData.TimeSeries[0].Usage.StorageBytes > 0 {
			fmt.Printf("syncing %s\n", sourceStorage)
			product, err := productsmodel.GetBestMatch(ctx, tx, sourceStorage, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			discount, err := discountsmodel.GetBestMatch(ctx, tx, sourceStorage, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			storageQuantity := float64(bucketMetricsData.TimeSeries[0].Usage.StorageBytes) / 1000 / 1000 / 1000
			storageFact := factsmodel.New(dateTime, query, tenant, category, product, discount, storageQuantity)
			_, err = factsmodel.Ensure(ctx, tx, storageFact)
			checkErrExit(err)
		}

		if bucketMetricsData.TimeSeries[0].Usage.SentBytes > 0 {
			fmt.Printf("syncing %s\n", sourceTrafficOut)
			product, err := productsmodel.GetBestMatch(ctx, tx, sourceTrafficOut, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			discount, err := discountsmodel.GetBestMatch(ctx, tx, sourceTrafficOut, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			trafficOutQuantity := float64(bucketMetricsData.TimeSeries[0].Usage.SentBytes) / 1000 / 1000 / 1000
			trafficOutFact := factsmodel.New(dateTime, query, tenant, category, product, discount, trafficOutQuantity)
			_, err = factsmodel.Ensure(ctx, tx, trafficOutFact)
			checkErrExit(err)
		}

		if bucketMetricsData.TimeSeries[0].Usage.Requests > 0 {
			fmt.Printf("syncing %s\n", sourceTrafficOut)
			product, err := productsmodel.GetBestMatch(ctx, tx, sourceRequests, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			discount, err := discountsmodel.GetBestMatch(ctx, tx, sourceRequests, bucketMetricsData.TimeSeries[0].Start)
			checkErrExit(err)
			requestsQuantity := float64(bucketMetricsData.TimeSeries[0].Usage.Requests) / 1000
			requestsFact := factsmodel.New(dateTime, query, tenant, category, product, discount, requestsQuantity)
			_, err = factsmodel.Ensure(ctx, tx, requestsFact)
			checkErrExit(err)
		}

		err = tx.Commit()
		checkErrExit(err)
	}

	os.Exit(0)
}
