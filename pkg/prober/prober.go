package prober

type Probe interface {
	SetHealthy()
	SetNotHealthy(err error)
	SetReady()
	SetNotReady(err error)
	IsReady() bool
	IsHealthy() bool
}
