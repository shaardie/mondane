package checkmanager

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	// database driver
	_ "github.com/go-sql-driver/mysql"
)

type repository interface {
	GetHTTPChecks(ctx context.Context) (*[]httpCheck, error)
	GetHTTPCheck(ctx context.Context, id int64) (*httpCheck, error)
	GetHTTPChecksByUser(ctx context.Context, id int64) (*[]httpCheck, error)
	CreateHTTPCheck(ctx context.Context, c *httpCheck) (int64, error)
	UpdateHTTPCheck(ctx context.Context, c *httpCheck) error
	DeleteHTTPCheck(ctx context.Context, id int64) error
	GetHTTPResults(ctx context.Context, id int64) (*[]httpResult, error)
	CreateHTTPResult(ctx context.Context, r *httpResult) (int64, error)
}

// sqlRepository fullfills the repository interface
type sqlRepository struct {
	db *sqlx.DB
}

// newSQLRepository returns a new repository
func newSQLRepository(dialect string, database string) (*sqlRepository, error) {
	res := &sqlRepository{}
	// Connect to database
	db, err := sqlx.Connect(dialect, database)
	if err != nil {
		return res, err
	}
	res.db = db
	return res, nil
}

func (s *sqlRepository) GetHTTPChecks(ctx context.Context) (*[]httpCheck, error) {
	c := &[]httpCheck{}
	err := s.db.SelectContext(ctx, c,
		`SELECT
			id, user_id, url
		FROM
			http_checks`)
	if err != nil {
		return nil, fmt.Errorf("unable to get http checks, %w", err)
	}
	return c, nil
}

func (s *sqlRepository) GetHTTPCheck(ctx context.Context, id int64) (*httpCheck, error) {
	c := &httpCheck{}
	err := s.db.GetContext(ctx, c,
		`SELECT
			id, user_id, url
		FROM
			http_checks
		WHERE
			id = ?`,
		id)
	if err != nil {
		return nil, fmt.Errorf("Unable to get http check %v, %w", id, err)
	}
	return c, nil
}

func (s *sqlRepository) GetHTTPChecksByUser(ctx context.Context, id int64) (*[]httpCheck, error) {
	cs := &[]httpCheck{}
	err := s.db.SelectContext(ctx, cs,
		`SELECT
			id, user_id, url
		FROM
			http_checks
		WHERE
			user_id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("unable to get http checks from user %v, %w", id, err)
	}
	return cs, err
}

func (s *sqlRepository) CreateHTTPCheck(ctx context.Context, c *httpCheck) (int64, error) {
	r, err := s.db.ExecContext(ctx,
		`INSERT INTO http_checks
			(user_id, url)
		VALUES (?, ?)`,
		c.UserID, c.URL)
	if err != nil {
		return 0, fmt.Errorf("unable to insert new check %v into database, %w", *c, err)
	}
	return r.LastInsertId()
}

func (s *sqlRepository) UpdateHTTPCheck(ctx context.Context, c *httpCheck) error {
	if c.URL == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE http_checks
		SET url = ?
		WHERE id = ?`, c.URL, c.ID)
	if err != nil {
		return fmt.Errorf("unable to update check %v, %w", c.ID, err)
	}
	return nil
}

func (s *sqlRepository) DeleteHTTPCheck(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM http_checks
		WHERE id = ?`,
		id)
	return err
}

func (s *sqlRepository) GetHTTPResults(ctx context.Context, id int64) (*[]httpResult, error) {
	rs := &[]httpResult{}
	err := s.db.SelectContext(ctx, rs,
		`SELECT
			id, timestamp, check_id, success, status_code, duration, error
		FROM
			http_results
		WHERE
			check_id = ?`,
		id)
	if err != nil {
		return nil, fmt.Errorf("Unable to get http results for check %v, %w", id, err)
	}
	return rs, nil
}

func (s *sqlRepository) CreateHTTPResult(ctx context.Context, r *httpResult) (int64, error) {
	o, err := s.db.ExecContext(ctx,
		`INSERT INTO http_results
			(timestamp, check_id, success, status_code, duration, error)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.Timestamp, r.CheckID, r.Success, r.StatusCode, r.Duration, r.Error)
	if err != nil {
		return 0, fmt.Errorf("unable to insert new result %v into database, %w", *r, err)
	}
	return o.LastInsertId()
}
