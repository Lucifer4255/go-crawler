package repository

import (
	"context"

	"go-crawler/internal/crawl/search"
	"go-crawler/internal/db"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Ensure Repository implements service.PageRepository (and optionally PageRepositoryWriter).
var _ service.PageRepositoryWriter = (*Repository)(nil)

func (r *Repository) UpsertPage(ctx context.Context, page *model.Page) (*model.Page, error) {
	jobID, err := uuidFromString(page.JobID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpsertPage(ctx, db.UpsertPageParams{
		JobID:       jobID,
		Url:         page.URL,
		Title:       pgtype.Text{String: page.Title, Valid: page.Title != ""},
		Html:        page.Html,
		TextContent: page.TextContent,
	})
	if err != nil {
		return nil, err
	}
	return pageFromDB(&row), nil
}

func (r *Repository) GetPagesByJobID(ctx context.Context, jobID string) ([]*model.Page, error) {
	uid, err := uuidFromString(jobID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetPagesByJobID(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Page, len(rows))
	for i := range rows {
		out[i] = pageFromDB(&rows[i])
	}
	return out, nil
}

func (r *Repository) CreatePage(ctx context.Context, page *model.Page) error {
	_, err := r.UpsertPage(ctx, page)
	return err
}

func pageFromDB(row *db.Page) *model.Page {
	p := &model.Page{
		ID:          int(row.ID),
		JobID:       uuid.UUID(row.JobID.Bytes).String(),
		URL:         row.Url,
		Html:        row.Html,
		TextContent: row.TextContent,
		FetchedAt:   row.FetchedAt.Time,
	}
	if row.Title.Valid {
		p.Title = row.Title.String
	}
	return p
}

func uuidFromString(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func (r *Repository) ListPagesForIndex(ctx context.Context) ([]search.Document, error) {
	rows, err := r.queries.ListPagesForIndex(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]search.Document, len(rows))
	for i := range rows {
		out[i] = search.Document{
			ID:   int(rows[i].ID),
			Text: rows[i].Title.String + " " + rows[i].TextContent,
		}
	}
	return out, nil
}
