package collector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type tlsResult struct {
	ID          uint           `gorm:"primaryKey" json:"-"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CheckID     uint           `json:"-"`
	Timestamp   time.Time      `json:"timestamp"`
	Success     bool           `json:"success"`
	TLSVersion  string         `json:"tls_version"`
	Duration    time.Duration  `json:"duration"`
	CipherSuite string         `json:"cipher_suite"`
	Expiry      time.Time      `json:"expiry"`
	Error       string         `json:"error"`
}

type tlsCheckRequest struct {
	Host string `json:"host"`
	Port uint   `json:"port"`
}

type tlsCheck struct {
	db *gorm.DB `gorm:"-" json:"-"`

	CheckID    uint           `gorm:"primaryKey;column:id" json:"id"`
	CreatedAt  time.Time      `json:"-"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	UserID     uuid.UUID      `gorm:"type:varchar(36);not null;" json:"-"`
	Host       string         `json:"host"`
	Port       uint           `json:"port"`
	TLSResults []tlsResult    `gorm:"ForeignKey:CheckID;References:CheckID;" json:"-"`
}

func (c *tlsCheck) ID() uint {
	return c.CheckID
}

func (c *tlsCheck) DoCheck(t time.Time) error {
	result := &tlsResult{
		Timestamp: t,
		CheckID:   c.ID(),
	}

	before := time.Now()
	conn, err := tls.Dial("tcp", fmt.Sprintf("%v:%v", c.Host, c.Port), nil)
	result.Duration = time.Now().Sub(before)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
	} else {
		defer conn.Close()
		result.Success = true
		state := conn.ConnectionState()
		result.TLSVersion = tlsVersionName(state.Version)
		result.Expiry = getCertExpiry(&state)
		result.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	}

	err = c.db.Create(result).Error
	if err != nil {
		return fmt.Errorf("unable to save result %v of check %v, %w", c, result, err)
	}

	return nil
}

func tlsVersionName(id uint16) string {
	switch id {
	case tls.VersionSSL30:
		return "SSL 3.0"
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "unkown"
	}
}

func getCertExpiry(state *tls.ConnectionState) time.Time {
	earliest := time.Time{}
	for _, cert := range state.PeerCertificates {
		if (earliest.IsZero() || cert.NotAfter.Before(earliest)) && !cert.NotAfter.IsZero() {
			earliest = cert.NotAfter
		}
	}
	return earliest
}

type collectortls struct {
	db        *gorm.DB
	logger    *zap.SugaredLogger
	scheduler *scheduler
}

func init() {
	factories["tls"] = newCollectortls
}

func newCollectortls(logger *zap.SugaredLogger, db *gorm.DB, scheduler *scheduler) (Collector, error) {
	ch := &collectortls{
		db:        db,
		logger:    logger,
		scheduler: scheduler,
	}

	err := db.AutoMigrate(&tlsCheck{}, &tlsResult{})
	if err != nil {
		return nil, fmt.Errorf("unable to migrate database model, %w", err)
	}

	var checks []tlsCheck
	if err := ch.db.Find(&checks).Error; err != nil {
		return nil, fmt.Errorf("unable to get tls checks from database, %w", err)
	}

	for i := range checks {
		checks[i].db = db
		if err := scheduler.Add(&checks[i]); err != nil {
			return nil, fmt.Errorf("unable schedule tls check %v, %w", checks[i], err)
		}
	}

	return ch, nil
}

func (ch *collectortls) Type() string {
	return "tls"
}

func (ch *collectortls) Create(ctx context.Context, userID uuid.UUID, r io.Reader) (interface{}, error) {
	request := &tlsCheckRequest{}
	err := json.NewDecoder(r).Decode(request)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JSON, %w", err)
	}

	// Add to database
	check := &tlsCheck{
		db:     ch.db,
		Host:   request.Host,
		Port:   request.Port,
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

func (ch *collectortls) ReadByUser(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	var checks []tlsCheck
	err := ch.db.WithContext(ctx).Find(&checks, tlsCheck{UserID: userID}).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get tls checks from database, %w", err)
	}
	return &checks, nil
}

func (ch *collectortls) Read(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error) {
	check := &tlsCheck{
		UserID:  userID,
		CheckID: id,
	}
	err := ch.db.WithContext(ctx).First(check).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get database entry for %v, %w", check, err)
	}
	return check, nil
}

func (ch *collectortls) ReadResults(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error) {
	check := &tlsCheck{
		UserID:  userID,
		CheckID: id,
	}
	err := ch.db.WithContext(ctx).Preload("TLSResults").First(check).Error
	if err != nil {
		return nil, fmt.Errorf("unable to get database entry for %v, %w", check, err)
	}
	return check.TLSResults, nil
}

func (ch *collectortls) Update(ctx context.Context, userID uuid.UUID, id uint, r io.Reader) (interface{}, error) {
	// Get Updates
	updates := &tlsCheckRequest{}
	err := json.NewDecoder(r).Decode(updates)
	if err != nil {
		return nil, fmt.Errorf("unable to parse json, %w", err)
	}
	check := &tlsCheck{
		CheckID: id,
		UserID:  userID,
	}
	err = ch.db.Model(check).Updates(&tlsCheck{
		Host: updates.Host,
		Port: updates.Port,
	}).Error
	if err != nil {
		return nil, fmt.Errorf("unable to update check %v with %v, %w", check, updates, err)
	}

	return check, nil

}

func (ch *collectortls) Delete(ctx context.Context, userID uuid.UUID, id uint) error {
	check := &tlsCheck{UserID: userID, CheckID: id}
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
