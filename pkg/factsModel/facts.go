package factsModel

import (
	"context"
	"fmt"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func GetByFact(ctx context.Context, tx *sqlx.Tx, fact *db.Fact) (*db.Fact, error) {
	var facts []db.Fact
	err := sqlx.SelectContext(ctx, tx, &facts,
		`SELECT facts.* FROM facts WHERE date_time_id = $1 AND query_id = $2 AND tenant_id = $3 AND category_id = $4 AND product_id = $5 AND discount_id = $6`,
		fact.DateTimeId, fact.QueryId, fact.TenantId, fact.CategoryId, fact.ProductId, fact.DiscountId)
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return nil, nil
	}
	return &facts[0], nil
}

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureFact *db.Fact) (*db.Fact, error) {
	fact, err := GetByFact(ctx, tx, ensureFact)
	fmt.Printf(">>> %v\n", fact)
	if err != nil {
		fmt.Printf("A\n")
		return nil, err
	}
	if fact == nil {
		fmt.Printf("B\n")
		fact, err = Create(tx, ensureFact)
		if err != nil {
			return nil, err
		}
	}
	fmt.Printf("C\n")
	return fact, nil
}

func Create(p db.NamedPreparer, in *db.Fact) (*db.Fact, error) {
	var category db.Fact
	err := db.GetNamed(p, &category,
		"INSERT INTO facts (date_time_id, query_id, tenant_id, category_id, product_id, discount_id, quantity) VALUES (:date_time_id, :query_id, :tenant_id, :category_id, :product_id, :discount_id, :quantity) RETURNING *", in)
	return &category, err
}

func New(dateTime *db.DateTime, query *db.Query, tenant *db.Tenant, category *db.Category, product *db.Product, discount *db.Discount, quanity float64) *db.Fact {
	return &db.Fact{
		DateTimeId: dateTime.Id,
		QueryId:    query.Id,
		TenantId:   tenant.Id,
		CategoryId: category.Id,
		ProductId:  product.Id,
		DiscountId: discount.Id,
		Quantity:   quanity,
	}
}
