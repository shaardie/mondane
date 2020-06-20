package worker

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
)

type dbWriter struct {
	db       *gorm.DB
	tx       *gorm.DB
	ticker   *time.Ticker
	finished chan bool
	results  chan interface{}
}

func newDBWriter(transactionInterval time.Duration, results chan interface{}, db *gorm.DB) *dbWriter {
	return &dbWriter{
		db:       db,
		ticker:   time.NewTicker(transactionInterval),
		finished: make(chan bool, 1),
		results:  results,
	}
}

func (t *dbWriter) start() {
	go t.run()
}

func (t *dbWriter) wait() {
	<-t.finished
}

func (t *dbWriter) run() {
	t.tx = t.db.Begin()
	for r := range t.results {
		if err := t.tx.Create(r).Error; err != nil {
			log.Printf("unable to create result in database, %v", err)
		}
		select {
		case <-t.ticker.C:
			if err := t.tx.Commit().Error; err != nil {
				log.Printf("Unable to commit Results, %v", err)
				t.tx.Rollback()
			}
			log.Println("Committed Results")
			t.tx = t.db.Begin()
		default:
			continue
		}
	}
	if err := t.tx.Commit().Error; err != nil {
		log.Printf("Unable to commit Results, %v", err)
		t.tx.Rollback()
	}
	t.finished <- true
}
