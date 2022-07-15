package categoriesModel

import (
	"context"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func GetBySource(ctx context.Context, tx *sqlx.Tx, source string) (*db.Category, error) {
	var categories []db.Category
	err := sqlx.SelectContext(ctx, tx, &categories, `SELECT categories.* FROM categories WHERE source = $1`, source)
	if err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		return nil, nil
	}
	return &categories[0], nil
}

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureCategory *db.Category) (*db.Category, error) {
	category, err := GetBySource(ctx, tx, ensureCategory.Source)
	if err != nil {
		return nil, err
	}
	if category == nil {
		category, err = Create(tx, ensureCategory)
		if err != nil {
			return nil, err
		}
	}
	return category, nil
}

func Create(p db.NamedPreparer, in *db.Category) (*db.Category, error) {
	var category db.Category
	err := db.GetNamed(p, &category,
		"INSERT INTO categories (source,target) VALUES (:source,:target) RETURNING *", in)
	return &category, err
}
