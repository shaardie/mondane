package httpcheck

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/jmoiron/sqlx"

	// database driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/shaardie/mondane/httpcheck/proto"
)

// check represents a http check from the database
type check struct {
	ID     uint64 `db:"id"`
	UserID uint64 `db:"user_id"`
	URL    string `db:"url"`
}

// result represents a result of a http check from the database
type result struct {
	ID        uint64    `db:"id"`
	CheckID   uint64    `db:"check_id"`
	Timestamp time.Time `db:"timestamp"`
	Success   bool      `db:"success"`
}

type repository interface {
	// getAll gets all checks
	getAll(ctx context.Context) (*[]check, error)
	// getCheck a check by id
	getCheck(context.Context, uint64) (*check, error)
	// get all checks by user id
	getChecksByUser(context.Context, uint64) (*[]check, error)
	// createCheck a new check
	createCheck(context.Context, *check) (*check, error)
	// deleteCheck deletes a existing check
	deleteCheck(context.Context, uint64) error

	// createResult creates a result for a specific check
	createResult(context.Context, *result, uint64) error
	// getResults gets results by check id
	getResults(context.Context, uint64) (*[]result, error)
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

// getAll gets all checks
func (s *sqlRepository) getAll(ctx context.Context) (*[]check, error) {
	c := &[]check{}
	err := s.db.SelectContext(ctx, c, "select * from http_checks")
	return c, err
}

// getCheck a check by id
func (s *sqlRepository) getCheck(ctx context.Context, id uint64) (*check, error) {
	c := &check{}
	err := s.db.GetContext(ctx, c, "select * from http_checks where id = ?", id)
	return c, err
}

// get all checks by user id
func (s *sqlRepository) getChecksByUser(ctx context.Context, id uint64) (*[]check, error) {
	c := &[]check{}
	err := s.db.SelectContext(ctx, c, "select * from http_checks where user_id = ?", id)
	return c, err
}

// createCheck a new check
func (s *sqlRepository) createCheck(ctx context.Context, c *check) (*check, error) {
	r, err := s.db.ExecContext(ctx, "insert into http_checks (user_id, url) values (?, ?)", c.UserID, c.URL)
	if err != nil {
		return &check{}, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return &check{}, err
	}
	if id < 0 {
		return &check{}, errors.New("id is negative")
	}
	return s.getCheck(ctx, uint64(id))
}

// deleteCheck deletes a existing check
func (s *sqlRepository) deleteCheck(ctx context.Context, id uint64) error {
	_, err := s.db.ExecContext(ctx, "delete from http_checks where id = ?", id)
	return err
}

// createResult creates a result for a specific check
func (s *sqlRepository) createResult(ctx context.Context, result *result, checkID uint64) error {
	_, err := s.db.ExecContext(
		ctx,
		"insert into http_check_results (timestamp, check_id, success) values (?, ?, ?)",
		result.Timestamp, result.CheckID, result.Success)
	return err
}

// getResults gets results by check id
func (s *sqlRepository) getResults(ctx context.Context, id uint64) (*[]result, error) {
	r := &[]result{}
	err := s.db.SelectContext(ctx, r, "select * from http_check_results where check_id = ?", id)
	return r, err
}

// marshalCheck is a helper function to create a database check from a grpc check
func marshalCheck(c *proto.Check) *check {
	return &check{
		ID:     c.Id,
		UserID: c.UserId,
		URL:    c.Url,
	}
}

// unmarshalCheck is a helper function to create a grpc check from a database check
func unmarshalCheck(c *check) *proto.Check {
	return &proto.Check{
		Id:     c.ID,
		UserId: c.UserID,
		Url:    c.URL,
	}
}

// Helper function to unmarschal a collections of checks
func unmarshalCheckCollection(cs *[]check) *proto.Checks {
	checks := make([]*proto.Check, len(*cs))
	for i, c := range *cs {
		checks[i] = unmarshalCheck(&c)
	}
	return &proto.Checks{Checks: checks}
}

// Helper function to unmarshal a result
func unmarshalResult(r *result) (*proto.Result, error) {
	t, err := ptypes.TimestampProto(r.Timestamp)
	if err != nil {
		return &proto.Result{}, err
	}
	return &proto.Result{
		Id:        r.ID,
		CheckId:   r.CheckID,
		Timestamp: t,
		Success:   r.Success,
	}, nil
}

// Helper function to unmarshal a collections of results
func unmarshalResultCollection(rs *[]result) (*proto.Results, error) {
	results := make([]*proto.Result, len(*rs))
	for i, r := range *rs {
		pr, err := unmarshalResult(&r)
		if err != nil {
			return &proto.Results{}, err
		}
		results[i] = pr
	}
	return &proto.Results{Results: results}, nil
}
