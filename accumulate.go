package main

import (
	"context"
	"fmt"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"os"
	"time"
)

type AccumulateKey struct {
	Query     string
	Zone      string
	Tenant    string
	Namespace string
	Start     time.Time
}

func (this AccumulateKey) String() string {
	return this.Query + ":" + this.Zone + ":" + this.Tenant + ":" + this.Namespace
}

func accumulate(ctx context.Context, date time.Time, cloudscaleClient *cloudscale.Client) (map[AccumulateKey]uint64, error) {
	bucketMetricsRequest := cloudscale.BucketMetricsRequest{Start: date, End: date}
	bucketMetrics, err := cloudscaleClient.Metrics.GetBucketMetrics(ctx, &bucketMetricsRequest)
	if err != nil {
		return nil, err
	}

	accumulated := make(map[AccumulateKey]uint64)

	for _, bucketMetricsData := range bucketMetrics.Data {
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

		sourceStorage := AccumulateKey{
			Query:     sourceQueryStorage,
			Zone:      zone,
			Tenant:    tenantStr,
			Namespace: namespace,
			Start:     bucketMetricsData.TimeSeries[0].Start,
		}
		sourceTrafficOut := AccumulateKey{
			Query:     sourceQueryTrafficOut,
			Zone:      zone,
			Tenant:    tenantStr,
			Namespace: namespace,
			Start:     bucketMetricsData.TimeSeries[0].Start,
		}
		sourceRequests := AccumulateKey{
			Query:     sourceQueryRequests,
			Zone:      zone,
			Tenant:    tenantStr,
			Namespace: namespace,
			Start:     bucketMetricsData.TimeSeries[0].Start,
		}

		accumulated[sourceStorage] += uint64(bucketMetricsData.TimeSeries[0].Usage.StorageBytes)
		accumulated[sourceTrafficOut] += uint64(bucketMetricsData.TimeSeries[0].Usage.SentBytes)
		accumulated[sourceRequests] += uint64(bucketMetricsData.TimeSeries[0].Usage.Requests)
	}

	return accumulated, nil
}
