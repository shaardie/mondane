// Package database holds the database definition and helper functions.
package database

import (
	"fmt"

	"github.com/jinzhu/gorm"

	// Database dialects
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// User represents a user in the database
type User struct {
	gorm.Model
	Email           string `gorm:"unique;not null" json:"email"`
	Password        []byte
	Firstname       string
	Surname         string
	Activated       bool
	ActivationToken string
	HostLimit       int `gorm:"not null"`
	Hosts           []Host
}

// Host represents a host in the database.
// Hosts belong to Users.
type Host struct {
	gorm.Model
	UserID   uint
	Ipv4     string
	Ipv6     string
	Hostname string
}

// ConnectDB connect to a database of the given dialect.
// Returns the database connection.
func ConnectDB(dialect string, url string) (*gorm.DB, error) {
	db, err := gorm.Open(dialect, url)
	if err != nil {
		return db, fmt.Errorf("unable to open database, %v", err)
	}

	err = db.AutoMigrate(&User{}, &Host{}).Error
	if err != nil {
		return db, fmt.Errorf("unable to migrate tables, %v", err)
	}

	return db, nil
}
