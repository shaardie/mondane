package db

import (
	"fmt"

	"github.com/joeshaw/envdecode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// config read from environment
type config struct {
	DBString string `env:"MONDANE_DATABASE,requireed"`
}

// NewDatabase returns a new repository
func NewDatabase() (*gorm.DB, error) {

	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		return nil, fmt.Errorf("unable to read config, %w", err)
	}

	db, err := gorm.Open(mysql.Open(c.DBString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(
		&User{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
