package prober

import "sync"

type combined struct {
	mu     sync.Mutex
	probes []Probe
}

func Combine(probes ...Probe) Probe {
	return &combined{probes: probes}
}

func (p *combined) SetReady() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		probe.SetReady()
	}
}

func (p *combined) SetNotReady(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		probe.SetNotReady(err)
	}
}

func (p *combined) IsReady() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		if !probe.IsReady() {
			return false
		}
	}
	return true
}

func (p *combined) SetHealthy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		probe.SetHealthy()
	}
}

func (p *combined) SetNotHealthy(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		probe.SetNotHealthy(err)
	}
}

func (p *combined) IsHealthy() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, probe := range p.probes {
		if !probe.IsHealthy() {
			return false
		}
	}
	return true
}
