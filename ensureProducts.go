package main

import (
	"database/sql"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

var (
	ensureProducts = []*db.Product{
		{
			Source: sourceQueryStorage + ":" + sourceZones[0],
			Target: sql.NullString{String: "1401", Valid: true},
			Amount: 0.0033,  // this is per DAY, equals 0.099 per GB per month
			Unit:   "GBDay", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryTrafficOut + ":" + sourceZones[0],
			Target: sql.NullString{String: "1403", Valid: true},
			Amount: 0.022,
			Unit:   "GB", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryRequests + ":" + sourceZones[0],
			Target: sql.NullString{String: "1405", Valid: true},
			Amount: 0.0055,
			Unit:   "KReq",
			During: db.InfiniteRange(),
		},
	}
)
