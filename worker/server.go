package worker

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/joeshaw/envdecode"

	"github.com/shaardie/mondane/checks"
	"github.com/shaardie/mondane/database"
)

type config struct {
	DatabaseDialect string `env:"MONDANE_WORKER_DATABASE_DIALECT,default=sqlite3"`
	Database        string `env:"MONDANE_WORKER_DATABASE,default=./mondane.db"`
	Listen          string `env:"MONDANE_WORKER_LISTEN,default=:8083"`
	Interval        int    `env:"MONDANE_WORKER_INTERVAL,default=10"`
}

type server struct {
	config *config
	db     *gorm.DB
}

func (s *server) run() error {
	// Make channels
	results := make(chan interface{}, 1024)
	interval := time.Duration(s.config.Interval) * time.Second
	var checker []checker

	// Signal handling
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Init http client
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// Connect to database
	if s.db == nil {
		db, err := database.ConnectDB(s.config.DatabaseDialect, s.config.Database)
		if err != nil {
			return fmt.Errorf("unable to connect to database, %v", err)
		}
		s.db = db
		log.Printf("Connected to database %v", s.config.Database)
	}

	// Get http checks from database
	var httpChecks []database.HTTPCheck
	if err := s.db.Find(&httpChecks).Error; err != nil {
		return fmt.Errorf("Unable to query database, %v", err)
	}
	log.Println("Got HTTP Checks from database")

	// Add http check to checker list
	for _, e := range httpChecks {
		checker = append(checker,
			newChecker(interval, results, checks.NewHTTPCheck(e, client)))
	}

	// Get tls checks from database
	var tlsChecks []database.TLSCheck
	if err := s.db.Find(&tlsChecks).Error; err != nil {
		return fmt.Errorf("Unable to query database, %v", err)
	}
	log.Println("Got TLS Checks from database")

	// Add tls checks to checker list
	for _, e := range tlsChecks {
		checker = append(checker, newChecker(interval, results, checks.NewTLSCheck(e)))
	}

	// Start Database Writer
	dbWriter := newDBWriter(5*time.Second, results, s.db)
	dbWriter.start()
	log.Println("Database Writer started")

	for _, c := range checker {
		c.startWitDelay(interval)
	}

	<-done
	log.Println("Signal received. Shutting down...")
	// Stop all
	for _, c := range checker {
		c.stop()
	}

	log.Println("Wait processes to stop")
	// Wait for stopped
	for _, c := range checker {
		c.wait()
	}

	log.Println("Stop database writer")
	close(results)
	log.Println("Wait for database writer to finish")
	<-dbWriter.finished
	return nil
}

// Run the mail server
func Run() error {
	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		return fmt.Errorf("unable to read config, %v", err)
	}
	s := server{
		config: &c,
	}
	return s.run()
}
