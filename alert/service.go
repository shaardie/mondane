package alert

// type config struct {
// 	Interval string `env:"MONDANE_ALERT_INTERVAL,required"`
// }

// type Service struct {
// 	logger  *zap.SugaredLogger
// 	db      *gorm.DB
// 	mail    *mail.Service
// 	ticker  *time.Ticker
// 	checker []interfaces.Checker
// }

// func New(logger *zap.SugaredLogger, db *gorm.DB, mail *mail.Service, checker []interfaces.Checker) *Service {
// 	return &Service{
// 		logger:  logger,
// 		db:      db,
// 		mail:    mail,
// 		ticker:  time.NewTicker(time.Minute),
// 		checker: checker,
// 	}
// }

// func (s *Service) Run() {
// 	s.logger.Info("Start alert service")
// 	for t := time.Now(); true; t = <-s.ticker.C {
// 		for _, c := range s.checker {
// 			_, err := c.Alert(context.Background())
// 			if err != nil {
// 				s.logger.Errorw("error while getting alertable checks", "checker", c, "error", err)
// 				continue
// 			}
// 		}
// 		s.logger.Info("Alert loop")
// 		var httpChecks []db.HTTPCheck
// 		result := s.db.Where(
// 			"success = ? AND failed_since < ? AND (reported is null or reported < ?)",
// 			db.StatusFailed, t.Add(-1*time.Minute), t.Add(-5*time.Minute),
// 		).Find(&httpChecks)
// 		if result.Error != nil {
// 			s.logger.Errorw("Error during alert query", "error", result.Error)
// 		}
// 		if result.RowsAffected == 0 {
// 			s.logger.Info("No results")
// 			continue
// 		}

// 		s.logger.Infow("Trigger", "checks", httpChecks)
// 	}
// }
