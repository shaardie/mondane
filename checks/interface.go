package checks

import "time"

type Check interface {
	Type() string
	ID() uint
	Check(time.Time) (interface{}, error)
}
