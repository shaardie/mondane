package collector

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	factories = make(map[string]func(logger *zap.SugaredLogger, db *gorm.DB, scheduler *scheduler) (Collector, error))
)

// New initializes the collectors
func New(logger *zap.SugaredLogger, db *gorm.DB, alerter Alerter) ([]Collector, error) {
	services := []Collector{}
	for key, f := range factories {
		service, err := f(logger, db, newScheduler(logger, alerter))
		if err != nil {
			return nil, fmt.Errorf("unable to create check service %v, %w", key, err)
		}
		services = append(services, service)
	}
	return services, nil
}
