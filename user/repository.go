package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	// database driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/shaardie/mondane/user/proto"
)

// user represent a user from the database
type user struct {
	ID              int64  `db:"id"`
	Email           string `db:"email"`
	Firstname       string `db:"firstname"`
	Surname         string `db:"surname"`
	Password        []byte `db:"password"`
	Activated       bool   `db:"activated"`
	ActivationToken string `db:"activation_token"`
}

// unmarshalUser is a helper function to create a grpc user from a database user
func unmarshalUser(u *user) *proto.User {
	return &proto.User{
		Id:        u.ID,
		Email:     u.Email,
		Firstname: u.Firstname,
		Surname:   u.Surname,
	}
}

// repository interface
type repository interface {
	// CRUD
	Create(ctx context.Context, u *user) (int64, error)
	Read(ctx context.Context, id int64) (*user, error)
	Update(ctx context.Context, u *user) error
	Delete(ctx context.Context, id int64) error

	// Activate user by token
	Activate(ctx context.Context, token string) error
	// Read by Mail
	ReadByMail(ctx context.Context, email string) (*user, error)
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

// Create a user
func (s *sqlRepository) Create(ctx context.Context, u *user) (int64, error) {
	// Insert new user
	r, err := s.db.ExecContext(ctx,
		`INSERT INTO users
			(email, firstname, surname, password, activated, activation_token)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.Email, u.Firstname, u.Surname, u.Password, u.Activated, u.ActivationToken)
	if err != nil {
		return 0, fmt.Errorf("unable to insert into users, %w", err)
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("unable to get id, %w", err)
	}
	return id, err
}

// Read a user by id
func (s *sqlRepository) Read(ctx context.Context, id int64) (*user, error) {
	u := &user{}
	err := s.db.GetContext(ctx, u,
		`SELECT
			id, email, firstname, surname, activated, password
		FROM
			users where id = ?`, id)
	return u, err
}

// Read a user by id
func (s *sqlRepository) ReadByMail(ctx context.Context, email string) (*user, error) {
	u := &user{}
	err := s.db.GetContext(ctx, u,
		`SELECT
			id, email, firstname, surname, activated, password
		FROM
			users where email = ?`, email)
	return u, err
}

// Update an existing user
func (s *sqlRepository) Update(ctx context.Context, u *user) error {
	// Update user in database
	_, err := s.db.ExecContext(ctx,
		`UPDATE users
		SET email = ?,
			firstname = ?,
			surname = ?,
			password = ?
		WHERE id = ?`,
		u.Email, u.Firstname, u.Surname,
		u.Password, u.ID)
	return err
}

// Delete a user by id.
func (s *sqlRepository) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM users WHERE id = ?",
		id)
	return err
}

// Activate a user by token
func (s *sqlRepository) Activate(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET activated = true WHERE activation_token = ?",
		token)
	return err
}
