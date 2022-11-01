package main

import "github.com/appuio/appuio-cloud-reporting/pkg/db"

var (
	ensureQueries = []*db.Query{
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
