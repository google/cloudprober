// Copyright 2018 Google Inc.
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

package udplistener

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
	"github.com/google/cloudprober/message"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets"
)

type serverConnStats struct {
	sync.Mutex
	msgCt map[string]int
}

// inputState contols the probes that run.
type inputState struct {
	seq           []int  // outgoing message seq# in order.
	src           string // outgoing message src.
	echoMode      bool   // controls whether server response to messages.
	statsInterval int32  // stats export interval (which resets counters).
	postTxSleep   string // duration to sleep after sending pkts.
}

const (
	localhost            = "localhost"
	interval             = time.Second
	timeout              = time.Second
	defaultServerType    = ProbeConf_DISCARD
	defaultStatsInterval = int32(3600000)
)

var (
	mZero = metrics.NewInt(0)
)

func sendPktsAndCollectReplies(ctx context.Context, t *testing.T, srvPort int, inp *inputState) []int {
	ctx, cancel := context.WithCancel(ctx)
	maxLen := 1024
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Fatalf("Starting UDP sender failed: %v", err)
	}
	t.Logf("Sender addr: %s", conn.LocalAddr().String())
	srvAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", srvPort))
	if err != nil {
		t.Fatalf("Error resolving udp addr for ':%d': %v", srvPort, err)
	}

	// set default host to locahost.
	src := localhost
	if inp.src != "" {
		src = inp.src
	}

	fm := message.NewFlowStateMap()
	fs := fm.FlowState(src, localhost)

	// Receive loop: keep receiving packets will context is cancelled.
	var rxSeq []int
	var wg sync.WaitGroup
	wg.Add(1)
	go func(rxctx context.Context) {
		defer wg.Done()
		b := make([]byte, maxLen)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn.SetReadDeadline(time.Now().Add(timeout))
			n, _, err := conn.ReadFromUDP(b)
			if err != nil {
				t.Logf("Error reading from udp: %v", err)
				continue
			}
			msg, err := message.NewMessage(b[:n])
			if err != nil {
				t.Logf("Error processing message: %v", err)
				continue
			}
			rxSeq = append(rxSeq, int(msg.Seq()))
		}
	}(ctx)

	// Send all pkts in a bunch and then get response.
	prevSeq := 0
	for _, seq := range inp.seq {
		if prevSeq != 0 && seq > prevSeq {
			time.Sleep(interval * time.Duration(seq-prevSeq))
		}
		fs.SetSeq(uint64(seq))
		buf, _, err := fs.CreateMessage(src, localhost, time.Now(), maxLen)
		if err != nil {
			t.Fatalf("Unable to create message: %v", err)
		}
		conn.SetWriteDeadline(time.Now().Add(timeout))
		if _, err := conn.WriteToUDP(buf, srvAddr); err != nil {
			t.Fatalf("Unable to send message: %v", err)
		}
		prevSeq = seq
	}

	time.Sleep(interval)
	if inp.postTxSleep != "" {
		dur, err := time.ParseDuration(inp.postTxSleep)
		if err != nil {
			t.Errorf("Parse error in duration string: %v", inp.postTxSleep)
		} else {
			time.Sleep(dur)
		}
	}
	cancel()
	wg.Wait()
	conn.Close()
	return rxSeq
}

func runProbe(ctx context.Context, t *testing.T, inp *inputState) ([]int, chan probeutils.ProbeResult, *probeRunResult, probeErr) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sysvars.Init(&logger.Logger{}, nil)
	p := &Probe{}

	// Default server mode is echo.
	srvType := defaultServerType
	if inp.echoMode {
		srvType = ProbeConf_ECHO
	}
	// force stats to be kept without reset.
	statsInterval := defaultStatsInterval
	if inp.statsInterval != 0 {
		statsInterval = inp.statsInterval
	}
	opts := &options.Options{
		Targets:  targets.StaticTargets("localhost"),
		Interval: interval,
		Timeout:  timeout,
		ProbeConf: &ProbeConf{
			Port: proto.Int32(0),
			Type: &srvType,
			StatsExportIntervalMsec: proto.Int32(statsInterval),
		},
	}
	if err := p.Init("udplistener", opts); err != nil {
		t.Fatalf("Error initialzing UDP probe")
	}
	port := p.conn.LocalAddr().(*net.UDPAddr).Port

	p.targets = p.opts.Targets.List()
	resultsChan := make(chan probeutils.ProbeResult, 10)
	go p.probeLoop(ctx, resultsChan)
	time.Sleep(interval) // Wait for echo loop to be active.

	rxSeq := sendPktsAndCollectReplies(ctx, t, port, inp)
	cancel()

	return rxSeq, resultsChan, p.res[localhost], p.errs
}

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	inp := &inputState{
		seq:      []int{1, 2, 3, 4, 5},
		echoMode: true,
	}
	pkts := int64(len(inp.seq))
	rxSeq, _, res, errs := runProbe(ctx, t, inp)
	if !reflect.DeepEqual(rxSeq, inp.seq) {
		t.Errorf("Probe response seq mismatch: got %v want %v. Check echo mode.", rxSeq, inp.seq)
	}

	mPkts := metrics.NewInt(pkts)
	mPktsSuccess := metrics.NewInt(pkts - 1)
	wantRes := &probeRunResult{
		target:  res.target,
		total:   *mPkts,
		success: *mPktsSuccess,
		ipdUS:   res.ipdUS,
		lost:    *mZero,
		delayed: *mZero,
	}
	if !reflect.DeepEqual(wantRes, res) {
		t.Errorf("Results mismatch: got %v want %v", res.Metrics(), wantRes.Metrics())
	}
	if len(errs.invalidMsgErrs) != 0 || len(errs.missingTargets) != 0 {
		t.Errorf("Got invalidmsgs=%v missingtgts:%v, want empty error vars", errs.invalidMsgErrs, errs.missingTargets)
	}
}

func TestDiscards(t *testing.T) {
	ctx := context.Background()
	inp := &inputState{
		seq: []int{1, 2, 3},
	}
	pkts := int64(len(inp.seq))
	rxSeq, _, res, errs := runProbe(ctx, t, inp)
	if len(rxSeq) != 0 {
		t.Errorf("Probe response seq: got %v, want empty array. Check discard mode.", rxSeq)
	}

	mPkts := metrics.NewInt(pkts)
	mPktsSuccess := metrics.NewInt(pkts - 1)
	wantRes := &probeRunResult{
		target:  res.target,
		total:   *mPkts,
		success: *mPktsSuccess,
		ipdUS:   res.ipdUS,
		lost:    *mZero,
		delayed: *mZero,
	}
	if !reflect.DeepEqual(wantRes, res) {
		t.Errorf("Results mismatch: got %v want %v", res.Metrics(), wantRes.Metrics())
	}
	if len(errs.invalidMsgErrs) != 0 || len(errs.missingTargets) != 0 {
		t.Errorf("Got invalidmsgs=%v missingtgts:%v, want empty error vars", errs.invalidMsgErrs, errs.missingTargets)
	}
}

func TestLoss(t *testing.T) {
	ctx := context.Background()
	// one out of order and one delayed message out of 5.
	inp := &inputState{
		seq: []int{1, 2, 4, 5, 3},
	}
	mOne := metrics.NewInt(1)
	pkts := int64(len(inp.seq))
	_, _, res, errs := runProbe(ctx, t, inp)

	mPkts := metrics.NewInt(pkts)
	mPktsSuccess := metrics.NewInt(pkts - 3)
	wantRes := &probeRunResult{
		target:  res.target,
		total:   *mPkts,
		success: *mPktsSuccess,
		ipdUS:   res.ipdUS,
		lost:    *mOne,
		delayed: *mOne,
	}
	if !reflect.DeepEqual(wantRes, res) {
		t.Errorf("Results mismatch: got %v want %v", res.Metrics(), wantRes.Metrics())
	}
	if len(errs.invalidMsgErrs) != 0 || len(errs.missingTargets) != 0 {
		t.Errorf("Got invalidmsgs=%v missingtgts:%v, want empty error vars", errs.invalidMsgErrs, errs.missingTargets)
	}
}

func TestUnknownSender(t *testing.T) {
	ctx := context.Background()
	src := "badhost"
	inp := &inputState{
		seq: []int{1, 2, 4, 5, 3},
		src: src,
	}
	_, _, res, errs := runProbe(ctx, t, inp)

	// Unknown sender => all metrics are zero.
	wantRes := &probeRunResult{
		target:  res.target,
		total:   *mZero,
		success: *mZero,
		ipdUS:   res.ipdUS,
		lost:    *mZero,
		delayed: *mZero,
	}
	if !reflect.DeepEqual(wantRes, res) {
		t.Errorf("Results mismatch: got %v want %v", res.Metrics(), wantRes.Metrics())
	}
	if len(errs.invalidMsgErrs) != 0 {
		t.Errorf("Invalidmsgs got %v want empty var", errs.invalidMsgErrs)
	}
	if errs.missingTargets[src] != len(inp.seq) {
		t.Errorf("MissingTargets[%s] got %v want %v", src, errs.missingTargets[src], len(inp.seq))
	}
}

func extractMetric(em *metrics.EventMetrics, key string) int64 {
	return em.Metric(key).(*metrics.Int).Int64()
}

// TestResultsChan export stats every 2 pkts and checks the results passed
// over the results channel.
func TestResultsChan(t *testing.T) {
	ctx := context.Background()

	inp := &inputState{
		seq:           []int{1, 2, 4, 5, 7, 6},
		statsInterval: int32(((4 * interval) / time.Millisecond).Nanoseconds()),
		postTxSleep:   "4s", // collect data for longer than pkts are sent.
	}
	_, resChan, _, _ := runProbe(ctx, t, inp)

	var res []probeutils.ProbeResult
readResChan:
	for {
		select {
		case <-time.After(1 * time.Second):
			break readResChan
		case r := <-resChan:
			t.Logf("Chan Res: %v", r.Metrics())
			res = append(res, r)
		}
	}

	// Test-0: >= 4 results from channel.
	minRes := 4
	if len(res) < minRes {
		t.Errorf("Too few results (%d < %d) from channel", len(res), minRes)
	}

	zeroPktsSeen := 0
	var lostVals []int64
	var delVals []int64
	for idx, r := range res {
		em := r.Metrics()
		// Test-1: All total values should be > 0 and <= 2.
		totCt := extractMetric(em, "total")
		minVal := int64(1)
		maxVal := int64(2)
		if totCt < minVal || totCt > maxVal {
			t.Errorf("(idx=%d) total val %d out of range [%d, %d]", idx, totCt, minVal, maxVal)
		}

		// Test-2: success + lost + delayed == total.
		// NOTE: for idx == 0, depending on timing, it is possible Test-3 fails.
		lostCt := extractMetric(em, "lost")
		delCt := extractMetric(em, "delayed")
		computeTot := extractMetric(em, "success") + lostCt + delCt
		if idx != 0 && computeTot > totCt {
			t.Errorf("(idx=%d) computed total (%d) > total val (%d)", idx, computeTot, totCt)
		}

		if idx != 0 && computeTot == 0 {
			zeroPktsSeen++
		}

		if lostCt != 0 {
			lostVals = append(lostVals, lostCt)
		}
		if delCt != 0 {
			delVals = append(delVals, delCt)
		}
	}

	// Test-3. Since we send only 6 packets (3 output intervals), we should see
	// atleast one interval with zero pkts seen.
	if zeroPktsSeen == 0 {
		t.Error("Expect at least one interval with zero incoming pkts.")
	}

	// Test-4: 2x lost packets, one per idx.
	wantLost := []int64{1, 1}
	if !reflect.DeepEqual(wantLost, lostVals) {
		t.Errorf("Lost count mismatch: got %v want %v", lostVals, wantLost)
	}

	// Test-5: 1x delayed packets.
	wantDel := []int64{1}
	if !reflect.DeepEqual(wantDel, delVals) {
		t.Errorf("Lost count mismatch: got %v want %v", delVals, wantDel)
	}
}
