package mail

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/joeshaw/envdecode"
	"gopkg.in/gomail.v2"
)

type config struct {
	Username string `env:"MONDANE_MAIL_USERNAME,required"`
	Password string `env:"MONDANE_MAIL_PASSWORD,required"`
	Server   string `env:"MONDANE_MAIL_SERVER,required"`
	Port     int    `env:"MONDANE_MAIL_HOST,default=25"`
	From     string `env:"MONDANE_MAIL_FROM,required"`
	Listen   string `env:"MONDANE_API_LISTEN,default=:8081"`
}

type server struct {
	dialer *gomail.Dialer
	config *config
	srv    *http.Server
	router *mux.Router
}

func (s *server) routes() {
	s.router = mux.NewRouter()
	s.router.Path("/api/sendmail/").Methods(http.MethodPost).HandlerFunc(s.sendMail())
	s.router.Use(logAccess)
	s.srv.Handler = s.router
	log.Println("routing finished")
}

// logAccess is a middleware which logs all requests
func logAccess(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now().Format(time.RFC3339)
		h.ServeHTTP(w, r)
		log.Printf("%v: %v %v %v %v %v %v",
			t, r.RemoteAddr, r.Host, r.UserAgent(), r.Method,
			r.URL.RequestURI(), r.Proto)
	})
}

func (s *server) sendMail() http.HandlerFunc {
	type requestMail struct {
		Recipient string `json:"recipient"`
		Subject   string `json:"subject"`
		Message   string `json:"message"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Get mail from request
		var mail requestMail
		err := json.NewDecoder(r.Body).Decode(&mail)
		if err != nil {
			http.Error(w, "broken json", http.StatusBadRequest)
			log.Printf("unable to parse request, %v", err)
			return
		}

		if mail.Recipient == "" {
			http.Error(w, "recipient required", http.StatusBadRequest)
			return
		}

		// New Message
		msg := gomail.NewMessage()
		msg.SetHeader("From", s.config.From)
		msg.SetHeader("Subject", mail.Subject)
		msg.SetHeader("To", mail.Recipient)
		msg.SetBody("text/plain", mail.Message)

		err = s.dialer.DialAndSend(msg)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			log.Printf("unable to send mail, %v", err)
			return
		}
		log.Printf("Sent mail to %v", mail.Recipient)
	}

}

// Run the mail server
func Run() error {
	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		return fmt.Errorf("unable to read config, %v", err)
	}

	s := server{
		dialer: gomail.NewDialer(c.Server, c.Port, c.Username, c.Password),
		config: &c,
		srv:    &http.Server{Addr: c.Listen},
	}
	s.routes()
	return s.srv.ListenAndServe()
}
