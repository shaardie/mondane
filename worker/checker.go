package worker

import (
	"log"
	"math/rand"
	"time"

	"github.com/shaardie/mondane/checks"
)

type checker struct {
	ticker   *time.Ticker
	results  chan interface{}
	check    checks.Check
	done     chan bool
	finished chan bool
}

func newChecker(checkInterval time.Duration, results chan interface{}, check checks.Check) checker {
	return checker{
		ticker:   time.NewTicker(checkInterval),
		results:  results,
		check:    check,
		done:     make(chan bool, 1),
		finished: make(chan bool, 1),
	}
}

func (c checker) startWitDelay(maxDelay time.Duration) {
	go func() {
		time.Sleep(time.Duration(rand.Int63n(maxDelay.Nanoseconds())))
		go c.run()
	}()
}

func (c checker) stop() {
	c.done <- true
}

func (c checker) wait() {
	<-c.finished
}

func (c checker) run() {
	log.Printf("Starting %v checks for %v", c.check.Type(), c.check.ID())
	for {
		select {
		case <-c.done:
			log.Printf("Shutting down %v checks for %v", c.check.Type(), c.check.ID())
			c.ticker.Stop()
			c.finished <- true
			log.Printf("Finished shutdown of %v checks for %v", c.check.Type(), c.check.ID())
			return
		case t := <-c.ticker.C:
			r, err := c.check.Check(t)
			if err != nil {
				log.Printf("Error during %v check for %v, %v", c.check.Type(), c.check.ID(), err)
			}
			select {
			case c.results <- r:
				continue
			default:
				log.Println("Queue is full")
				c.results <- r
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
