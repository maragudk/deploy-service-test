package sql

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/maragudk/errors"

	"github.com/maragudk/service/model"
)

// CreateJob to run immediately.
func (d *Database) CreateJob(ctx context.Context, name string, payload model.Map, timeout time.Duration) error {
	return d.inTransaction(ctx, func(tx *sqlx.Tx) error {
		return d.createJobInTx(ctx, tx, name, payload, timeout)
	})
}

func (d *Database) CreateJobForLater(ctx context.Context, name string, payload model.Map, timeout, after time.Duration) error {
	return d.inTransaction(ctx, func(tx *sqlx.Tx) error {
		return d.createJobForLaterInTx(ctx, tx, name, payload, timeout, after)
	})
}

func (d *Database) createJobInTx(ctx context.Context, tx *sqlx.Tx, name string, payload model.Map, timeout time.Duration) error {
	return d.createJobForLaterInTx(ctx, tx, name, payload, timeout, 0)
}

func (d *Database) createJobForLaterInTx(ctx context.Context, tx *sqlx.Tx, name string, payload model.Map, timeout, after time.Duration) error {
	if name == "" {
		panic("job name cannot be empty")
	}
	query := `insert into jobs (name, payload, timeout, run) values (?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, query, name, payload, timeout, model.Time{T: time.Now().Add(after)})
	return err
}

// GetJob which is eligible to run. Returns nil if no job available.
func (d *Database) GetJob(ctx context.Context) (*model.Job, error) {
	var job model.Job
	query := `
		update jobs
		set received = strftime('%Y-%m-%dT%H:%M:%fZ')
		where id = (
			select id from jobs
			where
				run <= strftime('%Y-%m-%dT%H:%M:%fZ') and (
					received is null or
					strftime('%Y-%m-%dT%H:%M:%fZ', received, (timeout/1000000000)||' second') <= strftime('%Y-%m-%dT%H:%M:%fZ')
				)
			order by created
			limit 1
		)
		returning *`
	if err := d.DB.GetContext(ctx, &job, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (d *Database) DeleteJob(ctx context.Context, id int) error {
	_, err := d.DB.ExecContext(ctx, `delete from jobs where id = ?`, id)
	return err
}
