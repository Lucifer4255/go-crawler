package repository

import (
	"context"
	"encoding/json"
	"errors"

	"go-crawler/internal/db"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ service.JobRepository = (*Repository)(nil)

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func dbJobToModel(j db.Job) (*model.CrawlJob, error) {
	var input model.CrawlInput
	if len(j.Input) > 0 {
		if err := json.Unmarshal(j.Input, &input); err != nil {
			return nil, err
		}
	}
	idStr := uuid.UUID(j.ID.Bytes).String()
	errStr := ""
	if j.Error.Valid {
		errStr = j.Error.String
	}
	return &model.CrawlJob{
		ID:           idStr,
		Input:        input,
		Status:       model.CrawlStatus(j.Status),
		PagesCrawled: int(j.PagesCrawled),
		Error:        errStr,
		CreatedAt:    j.CreatedAt.Time,
		UpdatedAt:    j.UpdatedAt.Time,
	}, nil
}

func (r *Repository) CreateJob(ctx context.Context, job *model.CrawlJob) error {
	inputJSON, err := json.Marshal(job.Input)
	if err != nil {
		return err
	}
	id, err := parseUUID(job.ID)
	if err != nil {
		return err
	}
	_, err = r.queries.CreateJob(ctx, db.CreateJobParams{
		ID:           id,
		Input:        inputJSON,
		Status:       string(job.Status),
		Error:        pgtype.Text{String: job.Error, Valid: job.Error != ""},
		PagesCrawled: int32(job.PagesCrawled),
		CreatedAt:    pgtype.Timestamptz{Time: job.CreatedAt, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: job.UpdatedAt, Valid: true},
	})
	return err
}

func (r *Repository) GetJob(ctx context.Context, id string) (*model.CrawlJob, error) {
	pid, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	j, err := r.queries.GetJob(ctx, pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("job not found")
		}
		return nil, err
	}
	return dbJobToModel(j)
}

func (r *Repository) UpdateJobStatus(ctx context.Context, id string, status model.CrawlStatus, errMsg string) error {
	pid, err := parseUUID(id)
	if err != nil {
		return err
	}
	_, err = r.queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		Status: string(status),
		Error:  pgtype.Text{String: errMsg, Valid: errMsg != ""},
		ID:     pid,
	})
	return err
}

func (r *Repository) TryIncrementPagesCrawled(ctx context.Context, id string, max int) (bool, error) {
	pid, err := parseUUID(id)
	if err != nil {
		return false, err
	}
	_, err = r.queries.TryIncrementPagesCrawled(ctx, db.TryIncrementPagesCrawledParams{
		ID:       pid,
		MaxPages: int32(max),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // max reached or job not found
		}
		return false, err
	}
	return true, nil
}
