package checks

import (
	"net/http"
	"time"

	"github.com/shaardie/mondane/database"
)

type HTTPCheck struct {
	client *http.Client
	check  database.HTTPCheck
}

func NewHTTPCheck(check database.HTTPCheck, client *http.Client) HTTPCheck {
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 10,
		}
	}
	return HTTPCheck{client: client, check: check}
}

func (HTTPCheck) Type() string {
	return "http"
}

func (hc HTTPCheck) ID() uint {
	return hc.check.ID
}

func (hc HTTPCheck) Check(t time.Time) (interface{}, error) {
	r := &database.HTTPResult{
		Time:        t,
		HTTPCheckID: hc.check.ID,
		Success:     false,
	}
	resp, err := hc.client.Get(hc.check.URL)
	if err != nil {
		return r, nil
	}
	resp.Body.Close()
	r.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return r, nil
}
