// Copyright 2020 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package grpc implements a gRPC probe.

This probes a cloudprober gRPC server and reports success rate, latency, and
validation failures.
*/
package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/common/oauth"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/grpc/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/endpoint"

	grpcprobepb "github.com/google/cloudprober/servers/grpc/proto"
	servicepb "github.com/google/cloudprober/servers/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
	grpcoauth "google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/resolver"

	// Import grpclb module so it can be used by name for DirectPath connections.
	_ "google.golang.org/grpc/balancer/grpclb"
)

const loadBalancingPolicy = `{"loadBalancingConfig":[{"grpclb":{"childPolicy":[{"pick_first":{}}]}}]}`

// TargetsUpdateInterval controls frequency of target updates.
var (
	TargetsUpdateInterval = 1 * time.Minute
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name     string
	src      string
	opts     *options.Options
	c        *configpb.ProbeConf
	l        *logger.Logger
	dialOpts []grpc.DialOption

	// Targets and cancellation function for each target.
	targets     []endpoint.Endpoint
	cancelFuncs map[string]context.CancelFunc
	targetsMu   sync.Mutex

	// Results by target.
	results map[string]*probeRunResult
}

// probeRunResult captures the metrics for a single target. Multiple threads
// can update metrics at the same time and the main thread periodically
// outputs the values in this struct.
type probeRunResult struct {
	sync.Mutex
	target        string
	total         metrics.Int
	success       metrics.Int
	latency       metrics.Value
	connectErrors metrics.Int
}

func (p *Probe) setupDialOpts() error {
	oauthCfg := p.c.GetOauthConfig()
	if oauthCfg != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(oauthCfg, p.l)
		if err != nil {
			return err
		}
		p.dialOpts = append(p.dialOpts, grpc.WithPerRPCCredentials(grpcoauth.TokenSource{oauthTS}))
	}
	altsCfg := p.c.GetAltsConfig()
	if altsCfg != nil {
		altsOpts := &alts.ClientOptions{
			TargetServiceAccounts:    altsCfg.GetTargetServiceAccount(),
			HandshakerServiceAddress: altsCfg.GetHandshakerServiceAddress(),
		}
		p.dialOpts = append(p.dialOpts, grpc.WithTransportCredentials(alts.NewClientCreds(altsOpts)))
	}

	if oauthCfg == nil && altsCfg == nil {
		p.dialOpts = append(p.dialOpts, grpc.WithInsecure())
	}
	p.dialOpts = append(p.dialOpts, grpc.WithDefaultServiceConfig(loadBalancingPolicy))
	p.dialOpts = append(p.dialOpts, grpc.WithBlock())
	return nil
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("not a gRPC probe config")
	}
	p.c = c
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.targets = p.opts.Targets.ListEndpoints()
	p.cancelFuncs = make(map[string]context.CancelFunc)
	p.src = sysvars.Vars()["hostname"]
	if err := p.setupDialOpts(); err != nil {
		return err
	}
	resolver.SetDefaultScheme("dns")
	return nil
}

func (p *Probe) updateTargetsAndStartProbes(ctx context.Context) {
	newTargets := p.opts.Targets.ListEndpoints()
	numNewTargets := len(newTargets)

	p.targetsMu.Lock()
	defer p.targetsMu.Unlock()
	if numNewTargets == 0 || numNewTargets < (len(p.targets)/2) {
		p.l.Errorf("Too few new targets, retaining old targets. New targets: %v, old count: %d", newTargets, len(p.targets))
		return
	}

	updatedTargets := make(map[string]string)
	defer func() {
		if len(updatedTargets) > 0 {
			p.l.Infof("Probe(%s) targets updated: %v", p.name, updatedTargets)
		}
	}()

	activeTargets := make(map[string]bool)
	// Create results structure and start probe loop for new targets.
	for _, tgtEp := range newTargets {
		tgt := net.JoinHostPort(tgtEp.Name, strconv.Itoa(tgtEp.Port))
		activeTargets[tgt] = true
		if _, ok := p.results[tgt]; ok {
			continue
		}
		updatedTargets[tgt] = "ADD"
		p.results[tgt] = p.newResult(tgt)
		probeCtx, probeCancelFunc := context.WithCancel(ctx)
		for i := 0; i < int(p.c.GetNumConns()); i++ {
			go p.oneTargetLoop(probeCtx, tgt, i, p.results[tgt])
		}
		p.cancelFuncs[tgt] = probeCancelFunc
	}

	// Stop probing for deleted targets by invoking cancelFunc.
	for tgt := range p.results {
		if activeTargets[tgt] {
			continue
		}
		p.cancelFuncs[tgt]()
		updatedTargets[tgt] = "DELETE"
		delete(p.results, tgt)
		delete(p.cancelFuncs, tgt)
	}
	p.targets = newTargets
}

// connectWithRetry attempts to connect to a target. On failure, it retries in
// an infinite loop until successful, incrementing connectErrors for every
// connection error. On success, it returns a client immediately.
// Interval between connects is controlled by connect_timeout_msec, defaulting
// to probe timeout.
func (p *Probe) connectWithRetry(ctx context.Context, tgt, msgPattern string, result *probeRunResult) *grpc.ClientConn {
	connectTimeout := p.opts.Timeout
	if p.c.GetConnectTimeoutMsec() > 0 {
		connectTimeout = time.Duration(p.c.GetConnectTimeoutMsec()) * time.Millisecond
	}
	var conn *grpc.ClientConn
	var err error
	for {
		select {
		case <-ctx.Done():
			p.l.Warningf("ProbeId(%s): context cancelled in connect loop.", msgPattern)
			return nil
		default:
		}
		connCtx, cancelFunc := context.WithTimeout(ctx, connectTimeout)
		conn, err = grpc.DialContext(connCtx, tgt, p.dialOpts...)
		cancelFunc()
		if err != nil {
			p.l.Warningf("ProbeId(%v) connect error: %v", msgPattern, err)
		} else {
			p.l.Infof("ProbeId(%v) connection established.", msgPattern)
			break
		}
		result.Lock()
		result.total.Inc()
		result.connectErrors.Inc()
		result.Unlock()
	}
	return conn
}

// oneTargetLoop connects to and then continuously probes a single target.
func (p *Probe) oneTargetLoop(ctx context.Context, tgt string, index int, result *probeRunResult) {
	msgPattern := fmt.Sprintf("%s,%s,%03d", p.src, tgt, index)

	conn := p.connectWithRetry(ctx, tgt, msgPattern, result)
	if conn == nil {
		return
	}
	defer conn.Close()

	client := servicepb.NewProberClient(conn)
	timeout := p.opts.Timeout
	method := p.c.GetMethod()

	msgSize := p.c.GetBlobSize()
	msg := make([]byte, msgSize)
	probeutils.PatternPayload(msg, []byte(msgPattern))
	ticker := time.NewTicker(p.opts.Interval)
	for {
		select {
		case <-ctx.Done():
			p.l.Warningf("ProbeId(%s): context cancelled in request loop.", msgPattern)
			ticker.Stop()
			return
		case <-ticker.C:
		}

		reqCtx, cancelFunc := context.WithTimeout(ctx, timeout)
		var success int64
		var delta time.Duration
		start := time.Now()
		var err error
		var peer peer.Peer
		opts := []grpc.CallOption{
			grpc.WaitForReady(true),
			grpc.Peer(&peer),
		}
		switch method {
		case configpb.ProbeConf_ECHO:
			req := &grpcprobepb.EchoMessage{
				Blob: []byte(msg),
			}
			_, err = client.Echo(reqCtx, req, opts...)
		case configpb.ProbeConf_READ:
			req := &grpcprobepb.BlobReadRequest{
				Size: proto.Int32(msgSize),
			}
			_, err = client.BlobRead(reqCtx, req, opts...)
		case configpb.ProbeConf_WRITE:
			req := &grpcprobepb.BlobWriteRequest{
				Blob: []byte(msg),
			}
			_, err = client.BlobWrite(reqCtx, req, opts...)
		default:
			p.l.Criticalf("Method %v not implemented", method)
		}
		cancelFunc()
		if err != nil {
			peerAddr := "unknown"
			if peer.Addr != nil {
				peerAddr = peer.Addr.String()
			}
			p.l.Warningf("ProbeId(%s) request failed: %v. ConnState: %v. Peer: %v", msgPattern, err, conn.GetState(), peerAddr)
		} else {
			success = 1
			delta = time.Since(start)
		}
		// TODO(ls692): add validators for probe result.
		result.Lock()
		result.total.Inc()
		result.success.AddInt64(success)
		result.latency.AddFloat64(delta.Seconds() / p.opts.LatencyUnit.Seconds())
		result.Unlock()
	}
}

func (p *Probe) newResult(tgt string) *probeRunResult {
	var latencyValue metrics.Value
	if p.opts.LatencyDist != nil {
		latencyValue = p.opts.LatencyDist.Clone()
	} else {
		latencyValue = metrics.NewFloat(0)
	}
	return &probeRunResult{
		target:  tgt,
		latency: latencyValue,
	}
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.results = make(map[string]*probeRunResult)
	p.updateTargetsAndStartProbes(ctx)

	ticker := time.NewTicker(p.opts.StatsExportInterval)
	defer ticker.Stop()

	targetsUpdateTicker := time.NewTicker(TargetsUpdateInterval)
	defer targetsUpdateTicker.Stop()

	for ts := range ticker.C {
		// Stop further processing and exit if context is canceled.
		// Same context is used by probe loops.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Output results.
		for targetName, result := range p.results {
			result.Lock()
			em := metrics.NewEventMetrics(ts).
				AddMetric("total", result.total.Clone()).
				AddMetric("success", result.success.Clone()).
				AddMetric("latency", result.latency.Clone()).
				AddMetric("connecterrors", result.connectErrors.Clone()).
				AddLabel("ptype", "grpc").
				AddLabel("probe", p.name).
				AddLabel("dst", targetName)
			result.Unlock()
			em.LatencyUnit = p.opts.LatencyUnit
			for _, al := range p.opts.AdditionalLabels {
				em.AddLabel(al.KeyValueForTarget(targetName))
			}
			p.opts.LogMetrics(em)
			dataChan <- em
		}

		// Finally, update targets and start new probe loops if necessary.
		// Executing this as the last step in the loop also ensures that new
		// targets have at least one cycle of probes before next output cycle.
		select {
		case <-targetsUpdateTicker.C:
			p.updateTargetsAndStartProbes(ctx)
		default:
		}
	}
}
