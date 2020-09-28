package alert

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/shaardie/mondane/collector"
	"github.com/shaardie/mondane/mail"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type alert struct {
	ID      uint      `gorm:"primaryKey"`
	UserID  uuid.UUID `json:"-"`
	CheckID uint      `json:"-"`
	Type    string    `json:"-"`

	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	LastSend    *time.Time     `json:"last_send"`
	FailedSince *time.Time     `json:"failed_since"`
}

type Service struct {
	logger *zap.SugaredLogger
	db     *gorm.DB
	mail   *mail.Service
}

func New(logger *zap.SugaredLogger, db *gorm.DB, mail *mail.Service) (*Service, error) {

	err := db.AutoMigrate(&alert{})
	if err != nil {
		return nil, fmt.Errorf("unable to migrate database model, %w", err)
	}

	return &Service{
		logger: logger,
		db:     db,
		mail:   mail,
	}, nil
}

func (s *Service) Trigger(check collector.Check, success bool) error {
	searchKey := alert{
		CheckID: check.GetID(),
		UserID:  check.GetUserID(),
		Type:    check.GetType(),
	}

	s.logger.Infow(
		"Triggered alert",
		"key", searchKey,
		"check", check,
		"success", success,
	)

	a := alert{}
	err := s.db.Where(searchKey).FirstOrCreate(&a).Error
	if err != nil {
		return fmt.Errorf("unable to get or create alert for check %v, %w", check, err)
	}

	update := alert{}
	now := time.Now()
	if success {
		update.FailedSince = nil
		update.FailedSince = nil
	} else {
		if a.FailedSince == nil {
			update.FailedSince = &now
		} else if a.FailedSince.Before(now.Add(-1*time.Minute)) &&
			(a.LastSend == nil || a.LastSend.Before(now.Add(-5*time.Minute))) {
			err = s.mail.SendFailure(context.Background(), check.FailureText(), check.GetUserID())
			if err != nil {
				return fmt.Errorf("failed to send mail on alert %v for check %v, %w", searchKey, check, err)
			}
			update.LastSend = &now
		}
	}

	err = s.db.Where(searchKey).Updates(update).Error
	if err != nil {
		return fmt.Errorf("failed to do updates %v on alert %v for check %v in database, %w", update, searchKey, check, err)
	}

	return nil
}
