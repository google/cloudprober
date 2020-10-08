package grpc

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/metrics/testutils"
	"github.com/google/cloudprober/probes/options"
	probepb "github.com/google/cloudprober/probes/proto"
	grpcpb "github.com/google/cloudprober/servers/grpc/proto"
	spb "github.com/google/cloudprober/servers/grpc/proto"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/targets/resolver"
	"google.golang.org/grpc"
)

var once sync.Once
var srvAddr string
var baseProbeConf = `
name: "grpc"
type: GRPC
targets {
	host_names: "%s"
}
interval_msec: 1000
timeout_msec: %d
grpc_probe {
	%s
	num_conns: %d
	connect_timeout_msec: 2000
}
`

func probeCfg(tgts, cred string, timeout, numConns int) (*probepb.ProbeDef, error) {
	conf := fmt.Sprintf(baseProbeConf, tgts, timeout, cred, numConns)
	cfg := &probepb.ProbeDef{}
	err := proto.UnmarshalText(conf, cfg)
	return cfg, err
}

type Server struct {
	delay time.Duration
	msg   []byte
}

// Echo reflects back the incoming message.
// TODO: return error if EchoMessage is greater than maxMsgSize.
func (s *Server) Echo(ctx context.Context, req *spb.EchoMessage) (*spb.EchoMessage, error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return req, nil
}

// BlobRead returns a blob of data.
func (s *Server) BlobRead(ctx context.Context, req *spb.BlobReadRequest) (*spb.BlobReadResponse, error) {
	return &spb.BlobReadResponse{
		Blob: s.msg[0:req.GetSize()],
	}, nil
}

// ServerStatus returns the current server status.
func (s *Server) ServerStatus(ctx context.Context, req *spb.StatusRequest) (*spb.StatusResponse, error) {
	return &spb.StatusResponse{
		UptimeUs: proto.Int64(42),
	}, nil
}

// BlobWrite returns the size of blob in the WriteRequest. It does not operate
// on the blob.
func (s *Server) BlobWrite(ctx context.Context, req *spb.BlobWriteRequest) (*spb.BlobWriteResponse, error) {
	return &spb.BlobWriteResponse{
		Size: proto.Int32(int32(len(req.Blob))),
	}, nil
}

// globalGRPCServer sets up runconfig and returns a gRPC server.
func globalGRPCServer() (string, error) {
	var err error
	once.Do(func() {
		var ln net.Listener
		ln, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return
		}
		grpcSrv := grpc.NewServer()
		srv := &Server{delay: time.Second / 2, msg: make([]byte, 1024)}
		grpcpb.RegisterProberServer(grpcSrv, srv)
		go grpcSrv.Serve(ln)
		srvAddr = ln.Addr().String()
		time.Sleep(time.Second * 2)
	})
	return srvAddr, err
}

// TestGRPCSuccess tests probe output on success.
// 2 connections, 1 probe/sec/conn, stats exported every 5 sec
// 	=> 5-10 results/interval. Test looks for minimum of 7 results.
func TestGRPCSuccess(t *testing.T) {
	addr, err := globalGRPCServer()
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}
	cfg, err := probeCfg(addr, "", 1000, 2)
	if err != nil {
		t.Fatalf("Error unmarshalling config: %v", err)
	}
	l := &logger.Logger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	iters := 5
	statsExportInterval := time.Duration(iters) * time.Second

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Timeout:             time.Second * 1,
		Interval:            time.Second * 1,
		ProbeConf:           cfg.GetGrpcProbe(),
		Logger:              l,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc-success", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 5)
	go p.Start(ctx, dataChan)
	time.Sleep(statsExportInterval * 2)
	found := false
	expectedLabels := map[string]string{
		"ptype": "grpc",
		"dst":   addr,
		"probe": "grpc-success",
	}

	for i := 0; i < 2; i++ {
		select {
		case em := <-dataChan:
			t.Logf("Probe results: %v", em.String())
			total := em.Metric("total").(*metrics.Int)
			success := em.Metric("success").(*metrics.Int)
			expect := int64(iters) + 2
			if total.Int64() < expect || success.Int64() < expect {
				t.Errorf("Got total=%d success=%d, expecting at least %d for each", total.Int64(), success.Int64(), expect)
			}
			gotLabels := make(map[string]string)
			for _, k := range em.LabelsKeys() {
				gotLabels[k] = em.Label(k)
			}
			if !reflect.DeepEqual(gotLabels, expectedLabels) {
				t.Errorf("Unexpected labels: got: %v, expected: %v", gotLabels, expectedLabels)
			}
			found = true
		default:
			time.Sleep(time.Second)
		}
	}
	if !found {
		t.Errorf("No probe results found")
	}
}

// TestConnectFailures attempts to connect to localhost:9 (discard port) and
// checks that stats are exported once every connect timeout.
// 2 connections, 0.5 connect attempt/sec/conn, stats exported every 6 sec
//  => 3 - 6 connect errors/sec. Test looks for minimum of 4 attempts.
func TestConnectFailures(t *testing.T) {
	addr := "localhost:9"
	cfg, err := probeCfg(addr, "", 1000, 2)
	if err != nil {
		t.Fatalf("Error unmarshalling config: %v", err)
	}
	l := &logger.Logger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	iters := 6
	statsExportInterval := time.Duration(iters) * time.Second

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Timeout:             time.Second * 1,
		Interval:            time.Second * 1,
		ProbeConf:           cfg.GetGrpcProbe(),
		Logger:              l,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc-connectfail", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 5)
	go p.Start(ctx, dataChan)
	time.Sleep(statsExportInterval * 2)
	found := false
	for i := 0; i < 2; i++ {
		select {
		case em := <-dataChan:
			t.Logf("Probe results: %v", em.String())
			total := em.Metric("total").(*metrics.Int)
			success := em.Metric("success").(*metrics.Int)
			connectErrs := em.Metric("connecterrors").(*metrics.Int)
			expect := int64(iters/2) + 1
			if success.Int64() > 0 {
				t.Errorf("Got %d probe successes, want all failures", success.Int64())
			}
			if total.Int64() < expect || connectErrs.Int64() < expect {
				t.Errorf("Got total=%d connectErrs=%d, expecting at least %d for each", total.Int64(), connectErrs.Int64(), expect)
			}
			found = true
		default:
			time.Sleep(time.Second)
		}
	}
	if !found {
		t.Errorf("No probe results found")
	}
}

func TestProbeTimeouts(t *testing.T) {
	addr, err := globalGRPCServer()
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}
	cfg, err := probeCfg(addr, "", 1000, 1)
	if err != nil {
		t.Fatalf("Error unmarshalling config: %v", err)
	}
	l := &logger.Logger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	iters := 5
	statsExportInterval := time.Duration(iters) * time.Second

	probeOpts := &options.Options{
		Targets:             targets.StaticTargets(addr),
		Timeout:             time.Millisecond * 100,
		Interval:            time.Second * 1,
		ProbeConf:           cfg.GetGrpcProbe(),
		Logger:              l,
		LatencyUnit:         time.Millisecond,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc-reqtimeout", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 5)
	go p.Start(ctx, dataChan)
	ems, err := testutils.MetricsFromChannel(dataChan, 2, statsExportInterval*3)
	if err != nil {
		t.Fatalf("Error retrieving metrics: %v", err)
	}
	mm := testutils.MetricsMap(ems)
	for target, vals := range mm["success"] {
		for _, v := range vals {
			success := v.Metric("success").(*metrics.Int)
			if success.Int64() > 0 {
				t.Errorf("Tgt %s unexpectedly succeeds, got=%d, want=0.", target, success.Int64())
				break
			}
		}
	}

	found := false
	for target, vals := range mm["total"] {
		prevTotal := int64(0)
		for _, v := range vals {
			total := v.Metric("total").(*metrics.Int)
			delta := total.Int64() - prevTotal
			// Even a single probe in iter is treated as success.
			if delta <= 0 {
				t.Errorf("Tgt %s did not get enough probes, got=%d, want>=1", target, delta)
				break
			}
			found = true
		}
	}
	if !found {
		t.Errorf("No probe results found")
	}
}

type testTargets struct {
	r *resolver.Resolver

	start        time.Time
	startTargets []endpoint.Endpoint

	switchDur   time.Duration
	nextTargets []endpoint.Endpoint
}

func newTargets(startTargets, nextTargets []endpoint.Endpoint, switchDur time.Duration) targets.Targets {
	return &testTargets{r: resolver.New(), startTargets: startTargets, nextTargets: nextTargets, start: time.Now(), switchDur: switchDur}
}

func (t *testTargets) ListEndpoints() []endpoint.Endpoint {
	if time.Since(t.start) > t.switchDur {
		return t.nextTargets
	}
	return t.startTargets
}

func (t *testTargets) Resolve(name string, ipVer int) (net.IP, error) {
	return t.r.Resolve(name, ipVer)
}

func sumIntMetrics(inp []*metrics.EventMetrics, metricName string) int64 {
	sum := metrics.NewInt(0)
	for _, em := range inp {
		sum.Add(em.Metric(metricName))
	}
	return sum.Int64()
}

func TestTargets(t *testing.T) {
	addr, err := globalGRPCServer()
	if err != nil {
		t.Fatalf("Error initializing global config: %v", err)
	}
	cfg, err := probeCfg(addr, "", 1000, 2)
	if err != nil {
		t.Fatalf("Error unmarshalling config: %v", err)
	}
	l := &logger.Logger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	goodTargets := endpoint.EndpointsFromNames([]string{addr})
	badTargets := endpoint.EndpointsFromNames([]string{"localhost:1", "localhost:2"})

	// Target discovery changes from good to bad targets after 2 statsExports.
	// And probe continues for 10 more stats exports.
	statsExportInterval := 1 * time.Second
	TargetsUpdateInterval = 2 * time.Second
	probeRunTime := 12 * time.Second

	probeOpts := &options.Options{
		Targets:             newTargets(goodTargets, badTargets, TargetsUpdateInterval-time.Second),
		Timeout:             time.Second,
		Interval:            time.Second * 1,
		ProbeConf:           cfg.GetGrpcProbe(),
		Logger:              l,
		LatencyUnit:         time.Millisecond,
		StatsExportInterval: statsExportInterval,
		LogMetrics:          func(em *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("grpc", probeOpts)
	dataChan := make(chan *metrics.EventMetrics, 10)
	go p.Start(ctx, dataChan)

	ems, err := testutils.MetricsFromChannel(dataChan, 12, probeRunTime)
	if err != nil {
		t.Fatalf("Error retrieving metrics: %v", err)
	}
	mm := testutils.MetricsMap(ems)

	connErrTargets := make(map[string]int64)
	connErrIterCount := 0
	for target, vals := range mm["connecterrors"] {
		s := sumIntMetrics(vals, "connecterrors")
		if s > 0 {
			connErrTargets[target] = s
		}
		if len(vals) > connErrIterCount {
			connErrIterCount = len(vals)
		}
	}

	successTargets := make(map[string]int64)
	successIterCount := 0
	for target, vals := range mm["success"] {
		s := sumIntMetrics(vals, "success")
		if s > 0 {
			successTargets[target] = s
			if connErrTargets[target] > 0 {
				t.Errorf("Target %s has both success and failures.", target)
			}
			if len(vals) > successIterCount {
				successIterCount = len(vals)
			}
		}
	}

	if len(successTargets) == 0 {
		t.Errorf("Got zero targets with success, want at least one.")
	}
	if len(connErrTargets) == 0 {
		t.Errorf("Got zero targets with connection errors, want at least one.")
	}
	if successIterCount >= connErrIterCount {
		t.Errorf("Got successIters(%d) >= connErrIters(%d), want '<'.", successIterCount, connErrIterCount)
	}
}
