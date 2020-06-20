// Package database holds the database definition and helper functions.
package database

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"

	// Database dialects
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// User represents a user in the database
type User struct {
	gorm.Model
	Email           string `gorm:"unique;not null"`
	Password        []byte
	Firstname       string
	Surname         string
	Activated       bool
	ActivationToken string
	HTTPChecks      []HTTPCheck
	TLSChecks       []TLSCheck
}

type HTTPCheck struct {
	gorm.Model
	UserID      uint
	URL         string `gorm:"not null"`
	LastUpdated time.Time
	Results     []HTTPResult
}

type HTTPResult struct {
	gorm.Model
	Time        time.Time `gorm:"not null"`
	HTTPCheckID uint
	Success     bool `gorm:"not null"`
}

type TLSCheck struct {
	gorm.Model
	UserID      uint
	Host        string `gorm:"not null"`
	Port        uint   `gorm:"not null"`
	LastUpdated time.Time
}

type TLSResult struct {
	gorm.Model
	TLSCheckID  uint
	Time        time.Time `gorm:"not null"`
	Success     bool      `gorm:"not null"`
	TLSVersion  uint16
	CipherSuite uint16
	Expiry      time.Time
	DialError   string
}

// ConnectDB connect to a database of the given dialect.
// Returns the database connection.
func ConnectDB(dialect string, url string) (*gorm.DB, error) {
	db, err := gorm.Open(dialect, url)
	if err != nil {
		return db, fmt.Errorf("unable to open database, %v", err)
	}

	err = db.AutoMigrate(
		&User{},
		&HTTPCheck{},
		&HTTPResult{},
		&TLSCheck{},
		&TLSResult{}).Error
	if err != nil {
		return db, fmt.Errorf("unable to migrate tables, %v", err)
	}

	return db, nil
}
