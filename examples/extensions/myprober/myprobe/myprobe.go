package myprobe

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/hoisie/redis"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	c       *ProbeConf
	targets []endpoint.Endpoint
	opts    *options.Options

	res map[string]*metrics.EventMetrics // Results by target
	l   *logger.Logger
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*ProbeConf)
	if !ok {
		return fmt.Errorf("not a my probe config")
	}
	p.c = c
	p.name = name
	p.opts = opts
	p.l = opts.Logger

	p.res = make(map[string]*metrics.EventMetrics)
	return nil
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	probeTicker := time.NewTicker(p.opts.Interval)

	for {
		select {
		case <-ctx.Done():
			probeTicker.Stop()
			return
		case <-probeTicker.C:
			// On probe tick, write data to the channel and run probe.
			for _, em := range p.res {
				dataChan <- em.Clone()
			}
			p.targets = p.opts.Targets.ListEndpoints()
			p.initProbeMetrics()
			probeCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(p.opts.Timeout))
			p.runProbe(probeCtx)
			cancelFunc()
		}
	}
}

// initProbeMetrics initializes missing probe metrics.
func (p *Probe) initProbeMetrics() {
	for _, target := range p.targets {
		if p.res[target.Name] != nil {
			continue
		}
		var latVal metrics.Value
		if p.opts.LatencyDist != nil {
			latVal = p.opts.LatencyDist.Clone()
		} else {
			latVal = metrics.NewFloat(0)
		}
		p.res[target.Name] = metrics.NewEventMetrics(time.Now()).
			AddMetric("total", metrics.NewInt(0)).
			AddMetric("success", metrics.NewInt(0)).
			AddMetric("latency", latVal).
			AddLabel("ptype", "redis").
			AddLabel("probe", p.name).
			AddLabel("dst", target.Name)
	}
}

// runProbeForTarget runs probe for a single target.
func (p *Probe) runProbeForTarget(ctx context.Context, target endpoint.Endpoint) error {
	client := &redis.Client{
		Addr: net.JoinHostPort(target.Name, strconv.Itoa(target.Port)),
	}
	key := p.c.GetKey()
	val := p.c.GetValue()

	switch p.c.GetOp() {
	case ProbeConf_SET:
		return client.Set(key, []byte(val))
	case ProbeConf_GET:
		_, err := client.Get(key)
		return err
	case ProbeConf_DELETE:
		_, err := client.Del(key)
		return err
	default:
		return fmt.Errorf("unknown op: %s", p.c.GetOp())
	}
}

// runProbe runs probe for all targets and update EventMetrics.
func (p *Probe) runProbe(ctx context.Context) {
	p.targets = p.opts.Targets.ListEndpoints()

	var wg sync.WaitGroup
	for _, target := range p.targets {
		wg.Add(1)

		go func(target endpoint.Endpoint, em *metrics.EventMetrics) {
			defer wg.Done()
			start := time.Now()
			em.Timestamp = start
			em.Metric("total").AddInt64(1)
			err := p.runProbeForTarget(ctx, target) // run probe just for a single target
			if err != nil {
				p.l.Errorf(err.Error())
				return
			}
			em.Metric("success").AddInt64(1)
			em.Metric("latency").AddFloat64(time.Now().Sub(start).Seconds() / p.opts.LatencyUnit.Seconds())
		}(target, p.res[target.Name])
	}

	wg.Wait()
}
