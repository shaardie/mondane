package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/jmoiron/sqlx"

	// database driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/shaardie/mondane/alert/proto"
)

// alert represents a alert from the database
type alert struct {
	ID         int64         `db:"id"`
	UserID     int64         `db:"user_id"`
	CheckID    int64         `db:"check_id"`
	CheckType  string        `db:"check_type"`
	SendMail   bool          `db:"send_mail"`
	LastSend   time.Time     `db:"last_send"`
	SendPeriod time.Duration `db:"send_period"`
}

// unmarshal alert to fit to protobuf
func unmarshalAlert(a *alert) (*proto.Alert, error) {
	lastSend, err := ptypes.TimestampProto(a.LastSend)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error, %w", err)
	}
	return &proto.Alert{
		Id:         a.ID,
		UserId:     a.UserID,
		CheckId:    a.CheckID,
		CheckType:  a.CheckType,
		SendMail:   a.SendMail,
		LastSend:   lastSend,
		SendPeriod: ptypes.DurationProto(a.SendPeriod),
	}, nil
}

// unmarshal a collection of alerts to fit to protobuf
func unmarshalAlerts(as *[]alert) (*proto.Alerts, error) {
	results := make([]*proto.Alert, len(*as))
	for i, r := range *as {
		pr, err := unmarshalAlert(&r)
		if err != nil {
			return nil, err
		}
		results[i] = pr
	}
	return &proto.Alerts{Alerts: results}, nil
}

// repository is the interface to the database
type repository interface {
	// Get an alert from its ids
	Get(context.Context, int64, int64) (*alert, error)
	// Get all alerts from a user id
	GetByUser(context.Context, int64) (*[]alert, error)
	// Get all alerts from a check by id and type
	GetByCheck(context.Context, int64, string) (*[]alert, error)
	// Create a new alert
	Create(context.Context, *alert) (*alert, error)
	// Delete a alert by id
	Delete(context.Context, int64, int64) error
	// Update last send from alert wit id
	UpdateLastSend(context.Context, int64) error
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
		return res, fmt.Errorf("unable to connect to %v database, %w", dialect, err)
	}
	res.db = db
	return res, nil
}

func (s *sqlRepository) Get(ctx context.Context, id int64, userID int64) (*alert, error) {
	alert := &alert{}
	err := s.db.GetContext(ctx, alert,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
 		FROM alerts
		WHERE id = ?
		AND user_id = ?`, id, userID)
	return alert, err
}

func (s *sqlRepository) GetByUser(ctx context.Context, userID int64) (*[]alert, error) {
	as := &[]alert{}
	err := s.db.SelectContext(ctx, as,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
		FROM alerts
		WHERE user_id = ?`, userID)
	return as, err
}

func (s *sqlRepository) GetByCheck(ctx context.Context, checkID int64, checkType string) (*[]alert, error) {
	as := &[]alert{}
	err := s.db.SelectContext(ctx, as,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
		FROM alerts
		WHERE check_id = ?
			AND check_type = ?`, checkID, checkType)
	return as, err
}

func (s *sqlRepository) Create(ctx context.Context, a *alert) (*alert, error) {
	r, err := s.db.ExecContext(ctx,
		`INSERT INTO alerts
			(user_id, check_id, check_type, send_mail, send_period, last_send)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.UserID, a.CheckID, a.CheckType, a.SendMail, a.SendPeriod, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("unable to create alert %w", err)
	}

	id, err := r.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("unable to get alert id, %w", err)
	}
	return s.Get(ctx, id, a.UserID)
}

func (s *sqlRepository) Delete(ctx context.Context, id int64, userID int64) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM alerts WHERE id = ? AND user_id = ?",
		id, userID)
	return err
}

func (s *sqlRepository) UpdateLastSend(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE alerts set last_send = ? WHERE id = ?", time.Now(), id)
	return err
}
