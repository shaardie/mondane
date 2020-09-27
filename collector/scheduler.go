package collector

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

type scheduler struct {
	storage      map[uint]*runner
	storageMutex *sync.Mutex
	logger       *zap.SugaredLogger
}

func newScheduler(logger *zap.SugaredLogger) *scheduler {
	return &scheduler{
		logger:       logger,
		storage:      make(map[uint]*runner),
		storageMutex: &sync.Mutex{},
	}
}

func (s *scheduler) Add(c check) error {
	s.storageMutex.Lock()
	defer s.storageMutex.Unlock()
	r := newRunner(s.logger, c)
	s.storage[c.ID()] = r
	r.start()
	return nil
}

func (s *scheduler) Remove(id uint) error {
	s.storageMutex.Lock()
	defer s.storageMutex.Unlock()
	mr, ok := s.storage[id]
	if !ok {
		return fmt.Errorf("check %v not started ", id)
	}

	mr.stop()
	mr.wait()
	delete(s.storage, id)
	return nil
}
