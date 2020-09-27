package main

import (
	"fmt"
	"log"
	"os"

	"github.com/shaardie/mondane/api"
	"github.com/shaardie/mondane/collector"
	"github.com/shaardie/mondane/mail"

	"github.com/shaardie/mondane/db"
	"go.uber.org/zap"
)

func mainWithError() error {
	// Logger
	baseLogger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Unable to initialize logger, %v", err)
		return err
	}
	logger := baseLogger.Sugar()
	logger.Info("Initialized logger")

	// Database
	db, err := db.NewDatabase()
	if err != nil {
		logger.Errorw("Unable to connect to the database", "error", err)
		return err
	}
	logger.Info("Connected to database")

	mail, err := mail.New(logger, db)
	if err != nil {
		logger.Errorw("Unable to create mail service", "error", err)
		return err
	}

	checkServices, err := collector.New(logger, db)
	if err != nil {
		logger.Errorw("Unable to create collectors", "error", err)
	}

	api, err := api.New(logger, db, checkServices, mail)
	if err != nil {
		logger.Errorw("Unable to start api", "error", err)
		return err
	}
	return api.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
