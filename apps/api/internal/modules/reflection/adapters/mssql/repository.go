package mssql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const selectColumns = `SELECT CONVERT(varchar(10),entry_date,23),body,created_at,updated_at FROM dbo.daily_reflections`

func scan(row *sql.Row) (application.Reflection, error) {
	var item application.Reflection
	err := row.Scan(&item.Date, &item.Text, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return item, httpx.ErrNotFound
	}
	return item, err
}
func (r *Repository) List(ctx context.Context, uid uuid.UUID) ([]application.Reflection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT TOP(30) CONVERT(varchar(10),entry_date,23),body,created_at,updated_at
        FROM dbo.daily_reflections WHERE user_id=CONVERT(uniqueidentifier,@p1) ORDER BY entry_date DESC`, uid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []application.Reflection{}
	for rows.Next() {
		var item application.Reflection
		if err := rows.Scan(&item.Date, &item.Text, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
func (r *Repository) Get(ctx context.Context, uid uuid.UUID, date string) (application.Reflection, error) {
	return scan(r.db.QueryRowContext(ctx, selectColumns+` WHERE user_id=CONVERT(uniqueidentifier,@p1) AND entry_date=CONVERT(date,@p2,23)`, uid.String(), date))
}
func (r *Repository) Save(ctx context.Context, uid uuid.UUID, date, body string) (application.Reflection, error) {
	// Serializing on the composite key avoids concurrent insert races without affecting other days/users.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return application.Reflection{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE dbo.daily_reflections WITH(UPDLOCK) SET body=@p3,updated_at=SYSUTCDATETIME()
      WHERE user_id=CONVERT(uniqueidentifier,@p1) AND entry_date=CONVERT(date,@p2,23)`, uid.String(), date, body)
	if err != nil {
		return application.Reflection{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return application.Reflection{}, err
	}
	if n == 0 {
		_, err = tx.ExecContext(ctx, `INSERT dbo.daily_reflections(user_id,entry_date,body)
          SELECT id,CONVERT(date,@p2,23),@p3 FROM dbo.users WHERE id=CONVERT(uniqueidentifier,@p1)`, uid.String(), date, body)
		if err != nil {
			return application.Reflection{}, err
		}
	}
	item, err := scan(tx.QueryRowContext(ctx, selectColumns+` WHERE user_id=CONVERT(uniqueidentifier,@p1) AND entry_date=CONVERT(date,@p2,23)`, uid.String(), date))
	if err != nil {
		return application.Reflection{}, err
	}
	if err := tx.Commit(); err != nil {
		return application.Reflection{}, err
	}
	return item, nil
}
func (r *Repository) Delete(ctx context.Context, uid uuid.UUID, date string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM dbo.daily_reflections WHERE user_id=CONVERT(uniqueidentifier,@p1)
      AND entry_date=CONVERT(date,@p2,23)`, uid.String(), date)
	return err
}
