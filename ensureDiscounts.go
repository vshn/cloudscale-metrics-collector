package main

import "github.com/appuio/appuio-cloud-reporting/pkg/db"

var (
	ensureDiscounts = []*db.Discount{
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
)
