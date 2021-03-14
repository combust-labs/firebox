package vmm

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/combust-labs/firebox/pkg/log"
)

type Ready struct {
	Metadata
}
type Unready struct {
	Metadata
}

type ReadinessActor struct {
	behavior actor.Behavior

	logger  *log.Logger
	manager *actor.PID
	spec    ProbeSpec

	counter int32
}

func NewReadinessActor(logger *log.Logger, manager *actor.PID, spec ProbeSpec) actor.Actor {
	act := &ReadinessActor{
		behavior: actor.NewBehavior(),
		logger:   logger,
		manager:  manager,
		spec:     spec,
	}
	act.behavior.Become(act.Unready)
	return act
}

func (a *ReadinessActor) Receive(context actor.Context) {
	a.behavior.Receive(context)
}

func (a *ReadinessActor) Unready(context actor.Context) {
	switch context.Message().(type) {
	case *Ready:
		a.counter++
		if a.counter >= a.spec.SuccessThreshold {
			a.logger.Infof("Becoming ready")
			a.behavior.Become(a.Ready)
			a.counter = 0
			context.Forward(a.manager)
		}
	}
}

func (a *ReadinessActor) Ready(context actor.Context) {
	switch context.Message().(type) {
	case *Unready:
		a.counter++
		if a.counter >= a.spec.FailureThreshold {
			a.logger.Infof("Becoming unready")
			a.behavior.Become(a.Unready)
			a.counter = 0
			context.Forward(a.manager)
		}
	}
}
