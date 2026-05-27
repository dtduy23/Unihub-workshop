package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"unihub-workshop/internal/model"
)

type ImportRepo struct {
	pool *pgxpool.Pool
}

func NewImportRepo(pool *pgxpool.Pool) *ImportRepo {
	return &ImportRepo{pool: pool}
}

func (r *ImportRepo) CreateJob(ctx context.Context, job *model.ImportJob) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO import_jobs (file_name, status, total_records) VALUES ($1, $2, $3) RETURNING id, started_at`,
		job.FileName, job.Status, job.TotalRecords,
	).Scan(&job.ID, &job.StartedAt)
}

func (r *ImportRepo) UpdateJob(ctx context.Context, job *model.ImportJob) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE import_jobs SET status=$1, total_records=$2, success_count=$3, error_count=$4, completed_at=CURRENT_TIMESTAMP WHERE id=$5`,
		job.Status, job.TotalRecords, job.SuccessCount, job.ErrorCount, job.ID)
	return err
}

func (r *ImportRepo) CreateError(ctx context.Context, e *model.ImportError) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO import_errors (job_id, row_number, raw_data, error_reason) VALUES ($1, $2, $3, $4)`,
		e.JobID, e.RowNumber, e.RawData, e.ErrorReason)
	return err
}

func (r *ImportRepo) FindAllJobs(ctx context.Context) ([]model.ImportJob, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, file_name, status, total_records, success_count, error_count, started_at, completed_at
		 FROM import_jobs ORDER BY started_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.ImportJob
	for rows.Next() {
		var j model.ImportJob
		if err := rows.Scan(&j.ID, &j.FileName, &j.Status, &j.TotalRecords,
			&j.SuccessCount, &j.ErrorCount, &j.StartedAt, &j.CompletedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *ImportRepo) FindErrorsByJobID(ctx context.Context, jobID string) ([]model.ImportError, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, job_id, row_number, raw_data, error_reason
		 FROM import_errors WHERE job_id=$1 ORDER BY row_number ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []model.ImportError
	for rows.Next() {
		var e model.ImportError
		if err := rows.Scan(&e.ID, &e.JobID, &e.RowNumber, &e.RawData, &e.ErrorReason); err != nil {
			return nil, err
		}
		errors = append(errors, e)
	}
	return errors, nil
}

func (r *ImportRepo) GetPool() *pgxpool.Pool {
	return r.pool
}
