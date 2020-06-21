package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	// database driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/shaardie/mondane/user/proto"
)

// user represent a user from the database
type user struct {
	ID              uint64 `db:"id"`
	Email           string `db:"email"`
	Firstname       string `db:"firstname"`
	Surname         string `db:"surname"`
	Password        []byte `db:"password"`
	Activated       bool   `db:"activated"`
	ActivationToken string `db:"activation_token"`
}

// repository interface
type repository interface {
	// get a user by id
	get(ctx context.Context, id uint64) (*user, error)
	// get a user by mail
	getByMail(ctx context.Context, email string) (*user, error)
	// activate a user by token
	activate(ctx context.Context, token string) error
	// new registers a new user
	new(ctx context.Context, u *user) (string, error)
	// update an existing user
	update(ctx context.Context, u *user) (*user, error)
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

// get a user by id
func (s *sqlRepository) get(ctx context.Context, id uint64) (*user, error) {
	u := &user{}
	err := s.db.GetContext(ctx, u, "select * from users where id = ?", id)
	return u, err
}

// get a user by mail
func (s *sqlRepository) getByMail(ctx context.Context, email string) (*user, error) {
	u := &user{}
	err := s.db.GetContext(ctx, u, "select * from users where email = ?", email)
	return u, err
}

// activate a user by token
func (s *sqlRepository) activate(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "update users set activated = true where activation_token = ?", token)
	return err
}

// new registers a new user
func (s *sqlRepository) new(ctx context.Context, u *user) (string, error) {
	// Check for mandatory keys
	if u.Email == "" || u.Password == nil {
		return "", errors.New("mandatory keys email and password")
	}

	// Generate hash from password
	password, err := bcrypt.GenerateFromPassword(u.Password, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	u.Password = password

	// Generate registration token
	token, err := generateToken(32)
	if err != nil {
		return token, fmt.Errorf("unable to generate token, %v", err)
	}

	// Insert new user
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users 
			(email, firstname, surname, password, activated, activation_token)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.Email, u.Firstname, u.Surname, u.Password, false, token)
	return token, err
}

// update an existing user
func (s *sqlRepository) update(ctx context.Context, u *user) (*user, error) {
	// Get user
	user, err := s.get(ctx, u.ID)
	if err != nil {
		return user, err
	}

	// Update user
	if u.Email != "" {
		user.Email = u.Email
	}
	if u.Firstname != "" {
		user.Firstname = u.Firstname
	}
	if u.Surname != "" {
		user.Surname = u.Surname
	}
	if u.Password != nil {
		user.Password = u.Password
	}

	// Update user in database
	_, err = s.db.ExecContext(ctx,
		"update users set email = ?, firstname = ?, surname = ?, password = ? WHERE id = ?",
		u.Email, u.Firstname, u.Surname, u.Password, u.ID)
	return user, err
}

// marshalUser is a helper function to create a database user from a grpc user
func marshalUser(u *proto.User) *user {
	return &user{
		ID:        u.Id,
		Email:     u.Email,
		Firstname: u.Firstname,
		Surname:   u.Surname,
		Password:  u.Password,
	}
}

// unmarshalUser is a helper function to create a grpc user from a database user
func unmarshalUser(u *user) *proto.User {
	return &proto.User{
		Id:        u.ID,
		Email:     u.Email,
		Firstname: u.Firstname,
		Surname:   u.Surname,
		Password:  u.Password,
	}
}
