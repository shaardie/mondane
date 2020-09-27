package db

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

// User represent a User from the database
type User struct {
	ID              uuid.UUID      `gorm:"type:varchar(36);primary_key"`
	CreatedAt       time.Time      `json:"-"`
	UpdatedAt       time.Time      `json:"-"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Email           string         `json:"email"`
	Firstname       string         `json:"firstname"`
	Surname         string         `json:"surname"`
	Password        []byte         `json:"-"`
	Activated       bool           `json:"-"`
	ActivationToken string         `json:"-"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (user *User) BeforeCreate(tx *gorm.DB) error {
	user.ID = uuid.NewV4()
	return nil
}
