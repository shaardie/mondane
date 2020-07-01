package checkmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/shaardie/mondane/checkmanager/proto"

	alert "github.com/shaardie/mondane/alert/proto"
	httpcheck "github.com/shaardie/mondane/httpcheck/proto"
)

type httpRunnerCheck struct {
	faiures   int
	httpCheck httpCheck
	db        repository
	alert     alert.AlertServiceClient
	httpcheck httpcheck.HTTPCheckServiceClient
}

func (hrc *httpRunnerCheck) CheckID() int64 {
	return hrc.httpCheck.ID
}

func (*httpRunnerCheck) CheckType() string {
	return "http"
}

func (hrc *httpRunnerCheck) DoCheck(ctx context.Context, t time.Time) error {
	r, err := hrc.httpcheck.Do(ctx, &httpcheck.Check{Url: hrc.httpCheck.URL})
	if err != nil {
		return fmt.Errorf("unable to do check via httpcheck service, %w", err)
	}
	result := &httpResult{
		CheckID:    hrc.httpCheck.ID,
		Duration:   r.Duration,
		Error:      r.Error,
		StatusCode: r.StatusCode,
		Success:    r.Success,
		Timestamp:  t,
	}
	_, err = hrc.db.CreateHTTPResult(ctx, result)
	if err != nil {
		return fmt.Errorf("unable to store new http check, %w", err)
	}

	if !r.Success {
		hrc.faiures++
	}

	if hrc.faiures > 3 {
		_, err = hrc.alert.Firing(ctx, &alert.Check{
			Id:   hrc.CheckID(),
			Type: hrc.CheckType(),
		})
		if err != nil {
			return fmt.Errorf("unable to fire alert %w", err)
		}
		hrc.faiures = 0
	}

	return nil
}

type httpCheck struct {
	ID     int64  `db:"id"`
	UserID int64  `db:"user_id"`
	URL    string `db:"url"`
}

func marshalHTTPCheck(c *proto.HTTPCheck) *httpCheck {
	return &httpCheck{
		ID:     c.Id,
		UserID: c.UserId,
		URL:    c.Url,
	}
}

func unmarshalHTTPCheck(c *httpCheck) *proto.HTTPCheck {
	return &proto.HTTPCheck{
		Id:     c.ID,
		UserId: c.UserID,
		Url:    c.URL,
	}
}

func unmarshalCheckCollection(cs *[]httpCheck) *proto.HTTPChecks {
	checks := make([]*proto.HTTPCheck, len(*cs))
	for i, c := range *cs {
		checks[i] = unmarshalHTTPCheck(&c)
	}
	return &proto.HTTPChecks{Checks: checks}
}

type httpResult struct {
	ID         int64     `db:"id"`
	CheckID    int64     `db:"check_id"`
	Timestamp  time.Time `db:"timestamp"`
	Success    bool      `db:"success"`
	StatusCode int64     `db:"status_code"`
	Duration   int64     `db:"duration"`
	Error      string    `db:"error"`
}

func marshalHTTPResult(c *proto.HTTPResult) (*httpResult, error) {
	t, err := ptypes.Timestamp(c.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal timestamp from %v, %w",
			c.String(), err)
	}
	return &httpResult{
		ID:         c.Id,
		CheckID:    c.CheckId,
		Timestamp:  t,
		Success:    c.Success,
		StatusCode: c.StatusCode,
		Duration:   c.Duration,
		Error:      c.Error,
	}, nil
}

func unmarshalHTTPResult(c *httpResult) (*proto.HTTPResult, error) {
	t, err := ptypes.TimestampProto(c.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal timestamp from %v, %w",
			*c, err)
	}
	return &proto.HTTPResult{
		Id:         c.ID,
		CheckId:    c.CheckID,
		Timestamp:  t,
		Success:    c.Success,
		StatusCode: c.StatusCode,
		Duration:   c.Duration,
		Error:      c.Error,
	}, nil
}

func unmarshalCheckResultCollection(cs *[]httpResult) (*proto.HTTPResults, error) {
	results := make([]*proto.HTTPResult, len(*cs))
	for i, c := range *cs {
		r, err := unmarshalHTTPResult(&c)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal %v in result collection, %w", c, err)
		}
		results[i] = r
	}
	return &proto.HTTPResults{Results: results}, nil
}
