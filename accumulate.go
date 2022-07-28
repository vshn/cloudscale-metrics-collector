package main

import (
	"context"
	"fmt"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"os"
	"time"
)

// AccumulateKey represents one data point ("fact") in the billing database.
// The actual value for the data point is not present in this type, as this type is just a map key, and the corresponding value is stored as a map value.
type AccumulateKey struct {
	Query     string
	Zone      string
	Tenant    string
	Namespace string
	Start     time.Time
}

// String returns the full "source" string as used by the appuio-cloud-reporting
func (this AccumulateKey) String() string {
	return this.Query + ":" + this.Zone + ":" + this.Tenant + ":" + this.Namespace
}

/*
accumulateBucketMetrics gets all the bucket metrics from cloudscale and puts them into a map. The map key is the "AccumulateKey",
and the value is the raw value of the data returned by cloudscale (e.g. bytes, requests). In order to construct the
correct AccumulateKey, this function needs to fetch the ObjectUsers's tags, because that's where the zone, tenant and
namespace are stored.
This method is "accumulating" data because it collects data from possibly multiple ObjectsUsers under the same
AccumulateKey. This is because the billing system can't handle multiple ObjectsUsers per namespace.
*/
func accumulateBucketMetrics(ctx context.Context, date time.Time, cloudscaleClient *cloudscale.Client) (map[AccumulateKey]uint64, error) {
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
		err = accumulateBucketMetricsForObjectsUser(accumulated, bucketMetricsData, objectsUser)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s: %v\n", bucketMetricsData.Subject.BucketName, err)
			continue
		}
	}

	return accumulated, nil
}

func accumulateBucketMetricsForObjectsUser(accumulated map[AccumulateKey]uint64, bucketMetricsData cloudscale.BucketMetricsData, objectsUser *cloudscale.ObjectsUser) error {
	if len(bucketMetricsData.TimeSeries) != 1 {
		return fmt.Errorf("There must be exactly one metrics data point, found %d", len(bucketMetricsData.TimeSeries))
	}

	tenantStr := objectsUser.Tags["tenant"]
	if tenantStr == "" {
		return fmt.Errorf("no tenant information found on objectsUser")
	}
	namespace := objectsUser.Tags["namespace"]
	if namespace == "" {
		return fmt.Errorf("no namespace information found on objectsUser")
	}
	zone := objectsUser.Tags["zone"]
	if zone == "" {
		return fmt.Errorf("no zone information found on objectsUser")
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

	return nil
}
