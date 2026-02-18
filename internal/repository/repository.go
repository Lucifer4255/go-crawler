package repository

import (
	"context"
	"fmt"
	"go-crawler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func New(ctx context.Context, connString string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected to database")
	if err := RunSchema(ctx, pool, schemaSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run schema: %w", err)
	}
	return &Repository{
		queries: db.New(pool),
		pool:    pool,
	}, nil
}

// RunSchema executes schema SQL (e.g. CREATE TABLE). Safe to call multiple times if schema uses IF NOT EXISTS.
func RunSchema(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	_, err := pool.Exec(ctx, schema)
	return err
}

// schemaSQL is the initial schema; must match internal/sql/schema/001_schema.sql.
const schemaSQL = `CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    input JSONB NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    pages_crawled INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
    id SERIAL PRIMARY KEY,
    job_id UUID REFERENCES jobs(id),
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    html TEXT NOT NULL,
    text_content TEXT NOT NULL,
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);`

func (r *Repository) Queries(ctx context.Context) *db.Queries {
	return r.queries
}

func (r *Repository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}
