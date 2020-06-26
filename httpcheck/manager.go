package httpcheck

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	alert "github.com/shaardie/mondane/alert/proto"
	"google.golang.org/grpc"
)

// httpCheckRunner is running a single http check on every interval
type httpCheckRunner struct {
	failed   int
	check    *check
	ticker   *time.Ticker
	db       repository
	client   *http.Client
	stopping chan bool
	stopped  chan bool
	alert    alert.AlertServiceClient
}

// newHTTPCheckRunner creates a new httpCheckRunner
func newHTTPCheckRunner(interval time.Duration, c *check, db repository,
	client *http.Client, alert alert.AlertServiceClient) *httpCheckRunner {
	return &httpCheckRunner{
		ticker:   time.NewTicker(interval),
		db:       db,
		stopped:  make(chan bool, 1),
		stopping: make(chan bool, 1),
		check:    c,
		client:   client,
		alert:    alert,
	}
}

// doCheck runs a check
func (cr *httpCheckRunner) doCheck(t time.Time) (*result, error) {
	r := &result{
		Timestamp: t,
		CheckID:   cr.check.ID,
		Success:   false,
	}
	resp, err := cr.client.Get(cr.check.URL)
	if err != nil {
		log.Println(err)
		return r, nil
	}
	resp.Body.Close()
	r.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return r, nil
}

// start the httpCheckRunner async
func (cr *httpCheckRunner) start() {
	go cr.run()
}

// stop the httpCheckRunner without waiting
func (cr *httpCheckRunner) stop() {
	cr.stopping <- true
}

// Wait for the httpCheckRunner to stop
func (cr *httpCheckRunner) wait() {
	<-cr.stopped
}

// run the httpCheckRunner
func (cr *httpCheckRunner) run() {
	for {
		select {
		case <-cr.stopping:
			cr.ticker.Stop()
			cr.stopped <- true
			return
		case t := <-cr.ticker.C:
			r, err := cr.doCheck(t)
			if err != nil {
				log.Printf("Check Failure for check %v, %v", cr.check.ID, err)
				break
			}

			err = cr.db.createResult(context.TODO(), r, cr.check.ID)
			if err != nil {
				log.Printf("Unable to write result of check %v, %v", cr.check.ID, err)
				break
			}

			// Reset alert
			if r.Success && cr.failed > 0 {
				cr.failed = 0
			}

			// Increase error count
			if !r.Success {
				cr.failed++

				// Trigger alert on 3 unsuccessful checks
				if cr.failed == 3 {
					cr.failed = 0
					_, err := cr.alert.Firing(context.TODO(), &alert.Check{
						Id:   cr.check.ID,
						Type: "http",
					})
					if err != nil {
						log.Printf("unable to trigger alert while checking check %v, %v",
							cr.check.ID, err)
						break
					}
				}
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// manager is an interface starting and stopping checks
type manager interface {
	start(*check) error
	stop(*check) error
	stopAll() error
}

// httpCheckManager is an implementation of the manager interface
type httpCheckManager struct {
	interval time.Duration
	db       repository
	checks   map[uint64]*httpCheckRunner
	client   *http.Client
	alert    alert.AlertServiceClient
}

// newCheckHTTPCheckManager creates a new httpCheckManager
func newCheckHTTPCheckManager(interval time.Duration, db repository, alertServer string) (*httpCheckManager, error) {
	d, err := grpc.Dial(alertServer, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to alert server, %v", err)
	}
	cm := &httpCheckManager{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		interval: interval,
		db:       db,
		checks:   make(map[uint64]*httpCheckRunner),
		alert:    alert.NewAlertServiceClient(d),
	}

	// Start all checks in the database
	checks, err := cm.db.getAll(context.TODO())
	if err != nil {
		return cm, err
	}
	// Looks more complicated than _, value := range *checks,
	// but &value will always point to the same latest value in the loop
	// and makes it difficult for us to use it.
	for i := range *checks {
		if err := cm.start(&(*checks)[i]); err != nil {
			return cm, err
		}
	}
	return cm, nil
}

// starts a new check
func (cm *httpCheckManager) start(c *check) error {
	cm.checks[c.ID] = newHTTPCheckRunner(
		cm.interval, c, cm.db, cm.client, cm.alert)
	cm.checks[c.ID].start()
	log.Printf("HTTP Check %v started", c.ID)
	return nil
}

// stops a check
func (cm *httpCheckManager) stop(c *check) error {
	r, existing := cm.checks[c.ID]
	if !existing {
		return fmt.Errorf("check with id %v is not running", c.ID)
	}
	r.stop()
	r.wait()

	log.Printf("HTTP Check %v stopped", c.ID)
	return nil
}

// stopAll checks
func (cm *httpCheckManager) stopAll() error {
	for _, v := range cm.checks {
		v.stop()
	}
	for _, v := range cm.checks {
		v.wait()
	}
	return nil
}
