package checkmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type check interface {
	CheckID() int64
	CheckType() string
	DoCheck(context.Context, time.Time) error
}

type memoryRunner struct {
	check    check
	ticker   *time.Ticker
	stopping chan bool
	stopped  chan bool
	logger   *zap.SugaredLogger
}

func newMemoryRunner(interval time.Duration, logger *zap.SugaredLogger, check check) *memoryRunner {
	return &memoryRunner{
		check:    check,
		ticker:   time.NewTicker(interval),
		stopping: make(chan bool, 1),
		stopped:  make(chan bool, 1),
		logger:   logger,
	}
}

// start the memoryRunner async
func (mr *memoryRunner) start() {
	mr.logger.Infow("Starting Memory Runner",
		"check id ", mr.check.CheckID(),
		"check type", mr.check.CheckType())
	go mr.run()
}

// stop the memoryRunner without waiting
func (mr *memoryRunner) stop() {
	mr.logger.Infow("Stopping Memory Runner",
		"check id ", mr.check.CheckID(),
		"check type", mr.check.CheckType())
	mr.stopping <- true
}

// Wait for the memoryRunner to stop
func (mr *memoryRunner) wait() {
	mr.logger.Infow("Wait for Memory Runner",
		"check id ", mr.check.CheckID(),
		"check type", mr.check.CheckType())
	<-mr.stopped
}

// run the memoryRunner
func (mr *memoryRunner) run() {
	mr.logger.Infow("Memory Runner started",
		"check id ", mr.check.CheckID(),
		"check type", mr.check.CheckType())
	for {
		select {
		case <-mr.stopping:
			mr.ticker.Stop()
			mr.stopped <- true
			mr.logger.Infow("Memory Runner stopped",
				"check id ", mr.check.CheckID(),
				"check type", mr.check.CheckType())
			return
		case t := <-mr.ticker.C:
			mr.logger.Infow("Memory Runner do check",
				"check id ", mr.check.CheckID(),
				"check type", mr.check.CheckType())
			err := mr.check.DoCheck(context.Background(), t)
			if err != nil {
				mr.logger.Errorw("Memory Runner check failed",
					"check id ", mr.check.CheckID(),
					"check type", mr.check.CheckType(),
					"error", err)
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

type checkKey struct {
	checkID   int64
	checkType string
}

type memoryManager struct {
	interval     time.Duration
	storage      map[checkKey]*memoryRunner
	storageMutex *sync.Mutex
	logger       *zap.SugaredLogger
}

func newMemoryManager(interval time.Duration, logger *zap.SugaredLogger) *memoryManager {
	return &memoryManager{
		interval:     interval,
		logger:       logger,
		storage:      make(map[checkKey]*memoryRunner),
		storageMutex: &sync.Mutex{},
	}
}

func (mm *memoryManager) start(c check) error {
	key := checkKey{
		checkID:   c.CheckID(),
		checkType: c.CheckType(),
	}

	mm.storageMutex.Lock()
	defer mm.storageMutex.Unlock()
	if _, ok := mm.storage[key]; ok {
		mm.logger.Errorw("key already exist in storage", "key", key)
		return fmt.Errorf("key already exist, %v", key)
	}
	mm.storage[key] = newMemoryRunner(mm.interval, mm.logger, c)
	mm.storage[key].start()
	return nil
}

func (mm *memoryManager) stop(c check) error {
	key := checkKey{
		checkID:   c.CheckID(),
		checkType: c.CheckType(),
	}
	mm.storageMutex.Lock()
	defer mm.storageMutex.Unlock()
	mr, ok := mm.storage[key]
	if !ok {
		mm.logger.Errorw("key do not exist in storage", "key", key)
		return fmt.Errorf("key do not exist, %v", key)
	}

	mr.stop()
	mr.wait()
	delete(mm.storage, key)
	return nil
}

func (mm *memoryManager) update(c check) error {
	key := checkKey{
		checkID:   c.CheckID(),
		checkType: c.CheckType(),
	}
	mm.storageMutex.Lock()
	defer mm.storageMutex.Unlock()
	mr, ok := mm.storage[key]
	if !ok {
		mm.logger.Errorw("key do not exist in storage", "key", key)
		return fmt.Errorf("key do not exist, %v", key)
	}

	mr.stop()
	mr.wait()
	delete(mm.storage, key)

	mm.storage[key] = newMemoryRunner(mm.interval, mm.logger, c)
	mm.storage[key].start()

	return nil
}

func (mm *memoryManager) stopAll() error {
	mm.logger.Info("Stopping all memory runner")
	for _, v := range mm.storage {
		v.stop()
	}

	mm.logger.Info("Waiting for memory runner to stop")
	for k, v := range mm.storage {
		v.wait()
		delete(mm.storage, k)
	}
	mm.logger.Info("Stopped all memory runner")
	return nil
}
