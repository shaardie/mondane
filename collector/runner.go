package collector

import (
	"time"

	"go.uber.org/zap"
)

type runner struct {
	check    Check
	ticker   *time.Ticker
	failed   bool
	stopping chan bool
	stopped  chan bool
	logger   *zap.SugaredLogger
	alerter  Alerter
}

func newRunner(logger *zap.SugaredLogger, check Check, alerter Alerter) *runner {
	return &runner{
		check:    check,
		ticker:   time.NewTicker(15 * time.Second),
		stopping: make(chan bool, 1),
		stopped:  make(chan bool, 1),
		logger:   logger,
		alerter:  alerter,
	}
}

// start the memoryRunner async
func (mr *runner) start() {
	mr.logger.Infow("Starting Memory Runner", "check", mr.check)
	go mr.run()
}

// stop the memoryRunner without waiting
func (mr *runner) stop() {
	mr.logger.Infow("Stopping Memory Runner", "check", mr.check)
	mr.stopping <- true
}

// Wait for the memoryRunner to stop
func (mr *runner) wait() {
	mr.logger.Infow("Wait for Memory Runner", "check", mr.check)
	<-mr.stopped
}

// run the memoryRunner
func (mr *runner) run() {
	mr.logger.Infow("Memory Runner started", "check", mr.check)

	for {
		select {
		case <-mr.stopping:
			mr.ticker.Stop()
			mr.stopped <- true
			mr.logger.Infow("Memory Runner stopped", "check", mr.check)
			return
		case t := <-mr.ticker.C:
			mr.logger.Infow("Memory Runner do check", "check", mr.check)
			success, err := mr.check.DoCheck(t)
			if err != nil {
				mr.logger.Errorw("Memory Runner check failed", "check", mr.check, "error", err)
				break
			}
			err = mr.alerter.Trigger(mr.check, success)
			if err != nil {
				mr.logger.Errorw("Memory Runner failed to trigger alert", "check", mr.check, "error", err)
				break
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
