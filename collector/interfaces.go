package collector

import (
	"context"
	"io"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Check interface {
	GetID() uint
	GetUserID() uuid.UUID
	GetType() string
	FailureText() string
	DoCheck(time.Time) (bool, error)
}

type Collector interface {
	Type() string

	Create(ctx context.Context, userID uuid.UUID, r io.Reader) (interface{}, error)
	ReadByUser(ctx context.Context, userID uuid.UUID) (interface{}, error)
	Read(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error)
	ReadResults(ctx context.Context, userID uuid.UUID, id uint) (interface{}, error)
	Update(ctx context.Context, userID uuid.UUID, id uint, r io.Reader) (interface{}, error)
	Delete(ctx context.Context, userID uuid.UUID, id uint) error
}

type Alerter interface {
	Trigger(check Check, success bool) error
}
