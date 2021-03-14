package ticker

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
)

type Start struct{}
type Stop struct{}
type HandlerFunc func()

type TickerActor struct {
	initialDelay time.Duration
	period       time.Duration
	handler      HandlerFunc
}

func (a *TickerActor) Receive(context actor.Context) {
	switch context.Message().(type) {
	case *Start:
		if a.initialDelay != 0 {
			context.SetReceiveTimeout(a.initialDelay)
		} else {
			a.handler()
			context.SetReceiveTimeout(a.period)
		}
	case *Stop:
		context.CancelReceiveTimeout()

	case *actor.ReceiveTimeout:
		a.handler()
		context.SetReceiveTimeout(a.period)
	}
}

type TickerActorOption func(*TickerActor)

func WithInitialDelay(initialDelay time.Duration) TickerActorOption {
	return func(a *TickerActor) {
		if initialDelay < 0 {
			panic(errors.New("negative initialDelay for TickerActor"))
		}
		a.initialDelay = initialDelay
	}
}

func NewTickerActor(period time.Duration, handler HandlerFunc, opts ...TickerActorOption) *TickerActor {
	if period <= 0 {
		panic(errors.New("non-positive period for NewTickerActor"))
	}
	if handler == nil {
		panic(errors.New("nil handler for NewTickerActor"))
	}
	a := &TickerActor{
		period:  period,
		handler: handler,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}
