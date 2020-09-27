package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"

	"github.com/joeshaw/envdecode"
	uuid "github.com/satori/go.uuid"
	"github.com/shaardie/mondane/db"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type config struct {
	Username string `env:"MONDANE_MAIL_USERNAME,required"`
	Password string `env:"MONDANE_MAIL_PASSWORD,required"`
	Server   string `env:"MONDANE_MAIL_SERVER,required"`
	Port     int    `env:"MONDANE_MAIL_HOST,default=25"`
	From     string `env:"MONDANE_MAIL_FROM,required"`
}

// Service struct to send mails
type Service struct {
	logger           *zap.SugaredLogger
	dialer           *gomail.Dialer
	config           *config
	registrationMail *template.Template
	db               *gorm.DB
}

// Mail respresent a mail to send
type Mail struct {
	Recipient string
	Subject   string
	Message   string
}

const (
	registrationMail = `
Hej {{.Firstname}},

please register to the Mondane Service by using the link below:

URL: {{.URL}}

Regards
`
)

func (s *Service) SendStatus(ctx context.Context, userID uuid.UUID) error {
	user := &db.User{}
	if err := s.db.WithContext(ctx).Preload("HTTPChecks").First(user, userID).Error; err != nil {
		return fmt.Errorf("unable to get user with id %v from database, %w", userID, err)
	}

	if err := s.SendMail(ctx, Mail{
		Recipient: user.Email,
		Subject:   "Mondane Status",
		Message:   "Somethings wrong",
	}); err != nil {
		return fmt.Errorf("unable to send mail, %w", err)
	}
	return nil
}

// SendRegistration sends a registration mail to new users
func (s *Service) SendRegistration(ctx context.Context, user *db.User, host string) error {
	var buf bytes.Buffer
	err := s.registrationMail.Execute(&buf, struct {
		Firstname string
		URL       string
	}{
		Firstname: user.Firstname,
		URL: fmt.Sprintf(
			"http://%v/api/v1/register?token=%v",
			host, user.ActivationToken),
	})
	if err != nil {
		return fmt.Errorf("unable to create template, %w", err)
	}

	err = s.SendMail(ctx, Mail{
		Recipient: user.Email,
		Subject:   "Mondane Registration",
		Message:   buf.String(),
	})
	if err != nil {
		return fmt.Errorf("unable to send mail, %w", err)
	}
	return nil
}

// SendMail send a mail
func (s *Service) SendMail(ctx context.Context, mail Mail) error {
	if mail.Recipient == "" {
		return errors.New("recipient empty")
	}

	// New Message
	msg := gomail.NewMessage()
	msg.SetHeader("From", s.config.From)
	msg.SetHeader("Subject", mail.Subject)
	msg.SetHeader("To", mail.Recipient)
	msg.SetBody("text/plain", mail.Message)

	// Dial and Send
	err := s.dialer.DialAndSend(msg)
	if err != nil {
		log.Printf("Failure sending mail: %v", err)
		return fmt.Errorf("unable to send mail, %v", err)
	}

	log.Printf("Sent mail to %v", mail.Recipient)
	return nil
}

// New creates a new Service instance
func New(logger *zap.SugaredLogger, db *gorm.DB) (*Service, error) {

	s := &Service{
		logger: logger,
		db:     db,
	}

	// Set Logger
	if logger == nil {
		return nil, errors.New("missing logger")
	}
	s.logger = logger

	// Set Config
	s.config = &config{}
	if err := envdecode.StrictDecode(s.config); err != nil {
		logger.Infow("Unable to read config",
			"error", err)
		return nil, fmt.Errorf("unable to read config, %w", err)
	}

	// Set dialer
	s.dialer = gomail.NewDialer(s.config.Server, s.config.Port, s.config.Username, s.config.Password)

	// Create templates
	s.registrationMail = template.Must(template.New("letter").Parse(registrationMail))

	return s, nil
}
