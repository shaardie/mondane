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
// It is user centered, so every call should filter by user.
type repository interface {
	Create(ctx context.Context, alert *alert) (*alert, error)
	Read(ctx context.Context, alertID, userID int64) (*alert, error)
	ReadAll(ctx context.Context, userID int64) (*[]alert, error)
	ReadByCheck(ctx context.Context, checkID, userID int64, checkType string) (*[]alert, error)
	Update(ctx context.Context, alert *alert) (*alert, error)
	Delete(ctx context.Context, alertID, userID int64) error
	UpdateLastSend(ctx context.Context, alertID, userID int64) error
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
	return s.Read(ctx, id, a.UserID)
}

func (s *sqlRepository) Read(ctx context.Context, alertID, userID int64) (*alert, error) {
	alert := &alert{}
	err := s.db.GetContext(ctx, alert,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
 		FROM alerts
		WHERE id = ?
		AND user_id = ?`, alertID, userID)
	return alert, err
}

func (s *sqlRepository) ReadAll(ctx context.Context, userID int64) (*[]alert, error) {
	as := &[]alert{}
	err := s.db.SelectContext(ctx, as,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
		FROM alerts
		WHERE user_id = ?`, userID)
	return as, err
}

func (s *sqlRepository) ReadByCheck(ctx context.Context, checkID, userID int64, checkType string) (*[]alert, error) {
	as := &[]alert{}
	err := s.db.SelectContext(ctx, as,
		`SELECT id, user_id, check_id, check_type, send_mail,last_send,
			send_period
		FROM alerts
		WHERE check_id = ?
			AND check_type = ?`, checkID, checkType)
	return as, err
}

func (s *sqlRepository) Update(ctx context.Context, alert *alert) (*alert, error) {
	r, err := s.db.ExecContext(ctx,
		`UPDATE alerts
		SET check_id = ?,
			check_type = ?,
			send_mail = ?,
			send_period = ?
		WHERE id = ? AND user_id = ?`,
		alert.CheckID, alert.CheckType, alert.SendMail,
		alert.SendPeriod, alert.ID, alert.UserID)
	if err != nil {
		return nil, fmt.Errorf("unable to update alert, %w", err)
	}
	if i, err := r.RowsAffected(); err != nil {
		return nil, fmt.Errorf(
			"unable to get affected rows while updating alert , %w", err)
	} else if i == 0 {
		return nil, fmt.Errorf(
			"no rows updated, no row with id=%v and user_id=%v", alert.ID, alert.UserID)
	} else if i > 1 {
		return nil, fmt.Errorf(
			"multiple rows affected from update with id=%v and user_id=%v",
			alert.ID, alert.UserID)
	}
	return s.Read(ctx, alert.ID, alert.UserID)

}
func (s *sqlRepository) Delete(ctx context.Context, alertID, userID int64) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM alerts WHERE id = ? AND user_id = ?",
		alertID, userID)
	return err
}

func (s *sqlRepository) UpdateLastSend(ctx context.Context, alertID, userID int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE alerts set last_send = ? WHERE id = ? AND user_id = ?",
		time.Now(), alertID, userID)
	return err
}
