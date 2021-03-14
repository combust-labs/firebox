package vmm

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/combust-labs/firebox/config"
	"github.com/combust-labs/firebox/pkg/actors/ticker"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/combust-labs/firebox/pkg/vmm"
	"net"
	"time"
)

type Metadata struct {
	ID string
	IP net.IP
}

type Start struct {
	Manager *actor.PID
}
type Started struct {
	Metadata
}

type Stop struct{}
type Stopped struct {
	ID string
}

type Failure struct {
	Err error
}

// internal message
type finished struct {
	err error
}

type VMMActor struct {
	behavior actor.Behavior

	logger  *log.Logger
	machine vmm.VMM

	manager *actor.PID
}

func NewVMMActor(logger *log.Logger, vmmConfig config.VMMConfig) actor.Actor {
	act := &VMMActor{
		behavior: actor.NewBehavior(),
		logger:   logger,
		machine:  vmm.NewVMM(logger, vmmConfig),
	}
	act.behavior.Become(act.Stopped)
	return act
}

func (a *VMMActor) Receive(context actor.Context) {
	a.behavior.Receive(context)
}

func (a *VMMActor) Stopped(context actor.Context) {
	switch msg := context.Message().(type) {
	case *Start:
		a.manager = msg.Manager
		err := a.startVMM(context, msg)
		if err != nil {
			context.Respond(&Failure{Err: err})
			return
		}
		a.behavior.Become(a.Started)
		context.Respond(&Started{a.metadata()})
	case *actor.Restarting:
		// TODO:
	}
}

func (a *VMMActor) metadata() Metadata {
	return Metadata{ID: a.machine.GetID(), IP: a.machine.GetIP()}
}

func (a *VMMActor) Started(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Stopping:
		a.stopVMM()
		a.behavior.Become(a.Stopped)
	case *Stop:
		a.stopVMM()
		a.behavior.Become(a.Stopped)
		context.Respond(&Stopped{ID: a.machine.GetID()})
	case *finished:
		a.logger.Warnf("VMM machine finished with error: %v", msg.err)
		context.Send(a.manager, &Stopped{ID: a.machine.GetID()})
		// lifecycle invokes *actor.Stopping
		context.Stop(context.Self())
	}
}

func (a *VMMActor) startVMM(context actor.Context, _ *Start) error {
	err := a.machine.Start()
	if err != nil {
		return err
	}
	a.logger.Infof("VMM IP %v", a.machine.GetIP())
	a.startHealthProbe(context)
	go func() {
		err = a.machine.WaitFinished()
		context.Send(context.Self(), &finished{
			err: err,
		})
	}()
	return nil
}

func (a *VMMActor) stopVMM() {
	a.logger.Infof("Stopping VMM")
	err := a.machine.Stop()
	if err != nil {
		a.logger.Warn("VMM stop error: %v", err)
	}
}

func (a *VMMActor) startHealthProbe(context actor.Context) {
	probeSpec := ProbeSpec{
		HTTPGet: HTTPGetAction{
			Scheme: "http",
			Host:   a.machine.GetIP().String(),
			Port:   8080,
			Path:   "/health",
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      3,
		PeriodSeconds:       1,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	// readiness actor
	props := actor.PropsFromProducer(func() actor.Actor {
		return NewReadinessActor(a.logger, a.manager, probeSpec)
	})
	readinessPID := context.SpawnPrefix(props, "vmm/readiness/")

	// health probe actor
	props = actor.PropsFromProducer(func() actor.Actor {
		return ticker.NewTickerActor(time.Duration(probeSpec.PeriodSeconds)*time.Second, TickerFunc(a.logger, probeSpec, context, readinessPID, a.metadata()))
	})
	healthPID := context.SpawnPrefix(props, "vmm/probe/")
	// start the probe
	context.Send(healthPID, &ticker.Start{})
}
