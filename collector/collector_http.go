package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type httpResult struct {
	ID         uint           `gorm:"primaryKey" json:"-"`
	CreatedAt  time.Time      `json:"-"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	CheckID    uint           `json:"-"`
	Timestamp  time.Time      `json:"timestamp"`
	Success    bool           `json:"success"`
	StatusCode int            `json:"status_code"`
	Duration   time.Duration  `json:"duration"`
	Error      string         `json:"error"`
}

type httpCheckRequest struct {
	URL string `json:"url"`
}

type httpCheck struct {
	client *http.Client `gorm:"-" json:"-"`
	db     *gorm.DB     `gorm:"-" json:"-"`

	CheckID     uint           `gorm:"primaryKey;column:id" json:"id"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      uuid.UUID      `gorm:"type:varchar(36);not null;" json:"-"`
	URL         string         `json:"url"`
	HTTPResults []httpResult   `gorm:"ForeignKey:CheckID;References:CheckID;" json:"-"`
}

func (c *httpCheck) ID() uint {
	return c.CheckID
}

func (c *httpCheck) DoCheck(t time.Time) error {
	result := &httpResult{
		Timestamp: t,
		CheckID:   c.ID(),
	}

	before := time.Now()
	resp, err := c.client.Get(c.URL)
	result.Duration = time.Now().Sub(before)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
	} else {
		result.StatusCode = resp.StatusCode
		result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	err = c.db.Create(result).Error
	if err != nil {
		return fmt.Errorf("unable to save result %v of check %v, %w", c, result, err)
	}

	return nil
}

type collectorHTTP struct {
	client    *http.Client
	db        *gorm.DB
	logger    *zap.SugaredLogger
	scheduler *scheduler
}

func init() {
	factories["http"] = newCollectorHTTP
}

func newCollectorHTTP(logger *zap.SugaredLogger, db *gorm.DB, scheduler *scheduler) (Collector, error) {
	ch := &collectorHTTP{
		client:    &http.Client{Timeout: 10 * time.Second},
		db:        db,
		logger:    logger,
		scheduler: scheduler,
	}

	err := db.AutoMigrate(&httpCheck{}, &httpResult{})
	if err != nil {
		return nil, fmt.Errorf("unable to migrate database model, %w", err)
	}

	var checks []httpCheck
	if err := ch.db.Find(&checks).Error; err != nil {
		return nil, fmt.Errorf("unable to get http checks from database, %w", err)
	}

	for i := range checks {
		checks[i].client = ch.client
		checks[i].db = db
		if err := scheduler.Add(&checks[i]); err != nil {
			return nil, fmt.Errorf("unable schedule http check %v, %w", checks[i], err)
		}
	}

	return ch, nil
}

func (ch *collectorHTTP) Type() string {
	return "http"
}

func (ch *collectorHTTP) Create(ctx context.Context, userID uuid.UUID, r io.Reader) (interface{}, error) {
	request := &httpCheckRequest{}
	err := json.NewDecoder(r).Decode(request)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON, %w", err)
	}

	// Add to database
	check := &httpCheck{
		db:     ch.db,
		client: ch.client,
		URL:    request.URL,
		UserID: userID,
	}

	err = ch.scheduler.Add(check)
	if err != nil {
		return nil, fmt.Errorf("unable to add new check %v to the scheduler, %w", check, err)
	}

	err = ch.db.WithContext(ctx).Create(check).Error
	if err != nil {
		return nil, fmt.Errorf("unable to create database entry, %w", err)
	}
	return check, nil
}

func (ch *collectorHTTP) ReadByUser(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	var checks []httpCheck
	err := ch.db.WithContext(ctx).Find(&checks, httpCheck{UserID: userID}).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get http checks from database, %w", err)
	}
	return &checks, nil
}

func (ch *collectorHTTP) Read(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error) {
	check := &httpCheck{
		UserID:  userID,
		CheckID: id,
	}
	err := ch.db.WithContext(ctx).First(check).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get database entry for %v, %w", check, err)
	}
	return check, nil
}

func (ch *collectorHTTP) ReadResults(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error) {
	check := &httpCheck{
		UserID:  userID,
		CheckID: id,
	}
	err := ch.db.WithContext(ctx).Preload("HTTPResults").First(check).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get database entry for %v, %w", check, err)
	}
	return check.HTTPResults, nil
}

func (ch *collectorHTTP) Update(ctx context.Context, userID uuid.UUID, id uint, r io.Reader) (interface{}, error) {
	// Get Updates
	updates := &httpCheckRequest{}
	err := json.NewDecoder(r).Decode(updates)
	if err != nil {
		return nil, fmt.Errorf("unable to parse json, %w", err)
	}
	check := &httpCheck{
		CheckID: id,
		UserID:  userID,
	}
	err = ch.db.Model(check).Updates(&httpCheck{URL: updates.URL}).Error
	if err != nil {
		return nil, fmt.Errorf("unable to update check %v with %v, %w", check, updates, err)
	}

	return check, nil

}

func (ch *collectorHTTP) Delete(ctx context.Context, userID uuid.UUID, id uint) error {
	check := &httpCheck{UserID: userID, CheckID: id}
	err := ch.db.WithContext(ctx).First(check).Error
	if err != nil {
		return fmt.Errorf("unable to get check %v from database, %w", check, err)
	}

	err = ch.db.WithContext(ctx).Delete(check).Error
	if err != nil {
		return fmt.Errorf("unable to remove check %v from database, %w", check, err)
	}

	err = ch.scheduler.Remove(id)
	if err != nil {
		return fmt.Errorf("unable to remove check %v from scheduler, %w", check, err)
	}

	return nil
}
