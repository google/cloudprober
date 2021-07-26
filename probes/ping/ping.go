// Copyright 2017-2021 The Cloudprober Authors.
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
Package ping implements a fast ping prober. It sends ICMP pings to a list of
targets and reports statistics on packets sent, received and latency
experienced.

This ping implementation supports two types of sockets: Raw and datagram ICMP
sockets.

Raw sockets require root privileges and all the ICMP noise is copied on all raw
sockets opened for ICMP. We have to deal with the unwanted ICMP noise.

On the other hand, datagram ICMP sockets are unprivileged and implemented in
such a way that kernel copies only relevant packets on them. Kernel assigns a
local port for such sockets and rewrites ICMP id of the outgoing packets to
match that port number. Incoming ICMP packets' ICMP id is matched with the
local port to find the correct destination socket.

More about these sockets: http://lwn.net/Articles/420800/
Note: On some linux distributions these sockets are not enabled by default; you
can enable them by doing something like the following:
      sudo sysctl -w net.ipv4.ping_group_range="0 5000"
*/
package ping

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/ping/proto"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/validators"
	"github.com/google/cloudprober/validators/integrity"
)

const (
	protocolICMP     = 1
	protocolIPv6ICMP = 58
	dataIntegrityKey = "data-integrity"
	icmpHeaderSize   = 8
	minPacketSize    = icmpHeaderSize + timeBytesSize // 16
	maxPacketSize    = 5000                           // MTU
)

type result struct {
	sent, rcvd        int64
	latency           metrics.Value
	validationFailure *metrics.Map
}

// icmpConn is an interface wrapper for *icmp.PacketConn to allow testing.
type icmpConn interface {
	read(buf []byte) (n int, peer net.Addr, recvTime time.Time, err error)
	write(buf []byte, peer net.Addr) (int, error)
	setReadDeadline(deadline time.Time)
	close()
}

// Probe implements a ping probe type that sends ICMP ping packets to the targets and reports
// back statistics on packets sent, received and the rtt.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	ipVer             int
	targets           []endpoint.Endpoint
	results           map[string]*result
	conn              icmpConn
	runCnt            uint64
	target2addr       map[string]net.Addr
	ip2target         map[[16]byte]string
	useDatagramSocket bool
	statsExportFreq   int // Export frequency
}

// Init initliazes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	p.name = name
	p.opts = opts
	if err := p.initInternal(); err != nil {
		return err
	}
	return p.listen()
}

// Helper function to initialize internal data structures, used by tests.
func (p *Probe) initInternal() error {
	c, ok := p.opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("no ping config")
	}
	p.c = c

	if p.l = p.opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}

	if p.c.GetPayloadSize() < timeBytesSize {
		return fmt.Errorf("payload_size (%d) cannot be smaller than %d", p.c.GetPayloadSize(), timeBytesSize)
	}
	if p.c.GetPayloadSize() > maxPacketSize-icmpHeaderSize {
		return fmt.Errorf("payload_size (%d) cannot be bigger than %d", p.c.GetPayloadSize(), maxPacketSize-icmpHeaderSize)
	}

	if err := p.configureIntegrityCheck(); err != nil {
		return err
	}

	p.statsExportFreq = int(p.opts.StatsExportInterval.Nanoseconds() / p.opts.Interval.Nanoseconds())
	if p.statsExportFreq == 0 {
		p.statsExportFreq = 1
	}

	// Unlike other probes, for ping probe, we need to know the IP version to
	// craft appropriate ICMP packets. We default to IPv4.
	p.ipVer = 4
	if p.opts.IPVersion != 0 {
		p.ipVer = p.opts.IPVersion
	}

	p.results = make(map[string]*result)
	p.ip2target = make(map[[16]byte]string)
	p.target2addr = make(map[string]net.Addr)
	p.useDatagramSocket = p.c.GetUseDatagramSocket()

	// Update targets run peiodically as well.
	p.updateTargets()

	return nil
}

// Adds an integrity validator if data integrity checks are not disabled.
func (p *Probe) configureIntegrityCheck() error {
	if p.c.GetDisableIntegrityCheck() {
		return nil
	}

	for _, v := range p.opts.Validators {
		if v.Name == dataIntegrityKey {
			p.l.Warningf("Not adding data-integrity validator as there is already a validator with the name \"%s\": %v", dataIntegrityKey, v)
			return nil
		}
	}

	iv, err := integrity.PatternNumBytesValidator(timeBytesSize, p.l)
	if err != nil {
		return err
	}

	v := &validators.Validator{
		Name:     dataIntegrityKey,
		Validate: func(input *validators.Input) (bool, error) { return iv.Validate(input.ResponseBody) },
	}

	p.opts.Validators = append(p.opts.Validators, v)

	return nil
}

func (p *Probe) listen() error {
	var err error
	p.conn, err = newICMPConn(p.opts.SourceIP, p.ipVer, p.useDatagramSocket)
	return err
}

func (p *Probe) updateTargets() {
	p.targets = p.opts.Targets.ListEndpoints()

	for _, target := range p.targets {
		for _, al := range p.opts.AdditionalLabels {
			al.UpdateForTarget(target)
		}

		// Update results map:
		p.updateResultForTarget(target.Name)

		ip, err := p.opts.Targets.Resolve(target.Name, p.ipVer)
		if err != nil {
			p.l.Warning("Bad target: ", target.Name, ". Err: ", err.Error())
			p.target2addr[target.Name] = nil
			continue
		}

		var a net.Addr
		if p.useDatagramSocket {
			a = &net.UDPAddr{IP: ip}
		} else {
			a = &net.IPAddr{IP: ip}
		}
		p.target2addr[target.Name] = a
		p.ip2target[ipToKey(ip)] = target.Name
	}
}

func (p *Probe) updateResultForTarget(t string) {
	if _, ok := p.results[t]; ok {
		return
	}

	var latencyValue metrics.Value
	if p.opts.LatencyDist != nil {
		latencyValue = p.opts.LatencyDist.Clone()
	} else {
		latencyValue = metrics.NewFloat(0)
	}

	p.results[t] = &result{
		latency:           latencyValue,
		validationFailure: validators.ValidationFailureMap(p.opts.Validators),
	}
}

// Match first 8-bits of the run ID with the first 8-bits of the ICMP sequence number.
// For raw sockets we also match ICMP id with the run ID. For datagram sockets, kernel
// does that matching for us. It rewrites the ICMP id of the outgoing packets with the
// fake local port selected at the time of the socket creation and uses the same criteria
// to forward incoming packets to the sockets.
func matchPacket(runID, pktID, pktSeq uint16, datagramSocket bool) bool {
	return runID>>8 == pktSeq>>8 && (datagramSocket || pktID == runID)
}

func (p *Probe) sendPackets(runID uint16, tracker chan bool) {
	seq := runID & uint16(0xff00)
	packetsSent := int32(0)

	// Allocate a byte buffer of the size: ICMP Header Size (8) + Payload Size
	// We re-use the same memory space for all outgoing packets.
	pktbuf := make([]byte, icmpHeaderSize+p.c.GetPayloadSize())

	for {
		for _, target := range p.targets {
			if p.target2addr[target.Name] == nil {
				p.l.Debug("Skipping unresolved target: ", target.Name)
				continue
			}
			p.prepareRequestPacket(pktbuf, runID, seq, time.Now().UnixNano())
			if _, err := p.conn.write(pktbuf, p.target2addr[target.Name]); err != nil {
				p.l.Warning(err.Error())
				continue
			}
			tracker <- true
			p.results[target.Name].sent++
		}

		packetsSent++
		if packetsSent >= p.c.GetPacketsPerProbe() {
			break
		}
		seq++
		time.Sleep(time.Duration(p.c.GetPacketsIntervalMsec()) * time.Millisecond)
	}
	close(tracker)
}

type rcvdPkt struct {
	id, seq uint16
	data    []byte
	tsUnix  int64
	target  string
}

// We use it to keep track of received packets in recvPackets.
type packetKey struct {
	target string
	seqNo  uint16
}

func (p *Probe) recvPackets(runID uint16, tracker chan bool) {
	// Number of expected packets: p.c.GetPacketsPerProbe() * len(p.targets)
	received := make(map[packetKey]bool, int(p.c.GetPacketsPerProbe())*len(p.targets))
	outstandingPkts := 0
	p.conn.setReadDeadline(time.Now().Add(p.opts.Timeout))
	pktbuf := make([]byte, maxPacketSize)
	for {
		// To make sure that we have picked up all the packets sent by the sender, we
		// use a tracker channel. Whenever sender successfully sends a packet, it notifies
		// on the tracker channel. Within receiver we keep a count of outstanding packets
		// and we increment this count  whenever there is an element on tracker channel.
		//
		// First run or we have received all the known packets so far. Check the tracker
		// to see if there is another packet to pick up.
		if outstandingPkts == 0 {
			if _, ok := <-tracker; ok {
				outstandingPkts++
			} else {
				// tracker has been closed and there are no outstanding packets.
				return
			}
		}

		// Read packet from the socket
		pktLen, peer, recvTime, err := p.conn.read(pktbuf)

		if err != nil {
			p.l.Warning(err.Error())
			// if it's a timeout, return immediately.
			if neterr, ok := err.(*net.OpError); ok && neterr.Timeout() {
				return
			}
			continue
		}
		if pktLen < minPacketSize {
			p.l.Warning("packet too small: size (", strconv.FormatInt(int64(pktLen), 10), ") < minPacketSize (16), from peer: ", peer.String())
			continue
		}

		// recvTime should never be zero:
		// -- On Unix systems, recvTime comes from the sockets.
		// -- On Non-Unix systems, read() call returns recvTime based on when
		//    packet was received by cloudprober.
		if recvTime.IsZero() {
			p.l.Info("didn't get fetch time from the connection (SO_TIMESTAMP), using current time")
			recvTime = time.Now()
		}

		var ip net.IP
		if p.useDatagramSocket {
			ip = peer.(*net.UDPAddr).IP
		} else {
			ip = peer.(*net.IPAddr).IP
		}
		target := p.ip2target[ipToKey(ip)]
		if target == "" {
			p.l.Debug("Got a packet from a peer that's not one of my targets: ", peer.String())
			continue
		}

		if !validEchoReply(p.ipVer, pktbuf[0]) {
			p.l.Warning("Not a valid ICMP echo reply packet from: ", target)
			continue
		}

		var pkt = rcvdPkt{
			tsUnix: recvTime.UnixNano(),
			target: target,
			// ICMP packet body starts from the 5th byte
			id:   binary.BigEndian.Uint16(pktbuf[4:6]),
			seq:  binary.BigEndian.Uint16(pktbuf[6:8]),
			data: pktbuf[8:pktLen],
		}

		rtt := time.Duration(pkt.tsUnix-bytesToTime(pkt.data)) * time.Nanosecond

		// check if this packet belongs to this run
		if !matchPacket(runID, pkt.id, pkt.seq, p.useDatagramSocket) {
			p.l.Info("Reply ", pkt.String(rtt), " Unmatched packet, probably from the last probe run.")
			continue
		}

		key := packetKey{pkt.target, pkt.seq}
		// Check if we have already seen this packet.
		if received[key] {
			p.l.Info("Duplicate reply ", pkt.String(rtt), " (DUP)")
			continue
		}
		received[key] = true

		// Read a "good" packet, where good means that it's one of the replies that
		// we were looking for.
		outstandingPkts--

		// Update probe result
		result := p.results[pkt.target]

		if p.opts.Validators != nil {
			failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: pkt.data}, result.validationFailure, p.l)

			// If any validation failed, return now, leaving the success and latency
			// counters unchanged.
			if len(failedValidations) > 0 {
				p.l.Debug("Target:", pkt.target, " ping.recvPackets: failed validations: ", strings.Join(failedValidations, ","), ".")
				continue
			}
		}

		result.rcvd++
		result.latency.AddFloat64(rtt.Seconds() / p.opts.LatencyUnit.Seconds())
	}
}

// Probe run ID is eventually used to identify packets of a particular probe run. To avoid
// assigning packets to the wrong probe run, it's important that we pick run id carefully:
//
// We use first 8 bits of the ICMP sequence number to match packets belonging to a probe
// run (rest of the 8-bits are used for actual sequencing of the packets). These first 8-bits
// of the sequence number are derived from the first 8-bits of the run ID. To make sure that
// nearby probe runs' sequence numbers don't have the same first 8-bits, we derive these bits
// from a running counter.
//
// For raw sockets, we have to also make sure (reduce the probablity) that two different probes
// (not just probe runs) don't get the same ICMP id (for datagram sockets that's not an issue as
// ICMP id is assigned by the kernel and kernel makes sure its unique). To achieve that we
// randomize the last 8-bits of the run ID and use run ID as ICMP id.
func (p *Probe) newRunID() uint16 {
	return uint16(p.runCnt)<<8 + uint16(rand.Intn(0x00ff))
}

// runProbe is called by Run for each probe interval. It does the following on
// each run:
//   * Resolve targets if target resolve interval has elapsed.
//   * Increment run count (runCnt).
//   * Get a new run ID.
//   * Starts a goroutine to receive packets.
//   * Send packets.
func (p *Probe) runProbe() {
	// Resolve targets if target resolve interval has elapsed.
	if (p.runCnt % uint64(p.c.GetResolveTargetsInterval())) == 0 {
		p.updateTargets()
	}
	p.runCnt++
	runID := p.newRunID()
	wg := new(sync.WaitGroup)
	tracker := make(chan bool, int(p.c.GetPacketsPerProbe())*len(p.targets))
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.recvPackets(runID, tracker)
	}()
	p.sendPackets(runID, tracker)
	wg.Wait()
}

// Start starts the probe and writes back the data on the provided channel.
// Probe should have been initialized with Init() before calling Start on it.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	if p.conn == nil {
		p.l.Critical("Probe has not been properly initialized yet.")
	}
	defer p.conn.close()

	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for ts := range ticker.C {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.runProbe()
		p.l.Debugf("%s: Probe finished.", p.name)
		if (p.runCnt % uint64(p.statsExportFreq)) != 0 {
			continue
		}
		for _, target := range p.targets {
			result := p.results[target.Name]
			em := metrics.NewEventMetrics(ts).
				AddMetric("total", metrics.NewInt(result.sent)).
				AddMetric("success", metrics.NewInt(result.rcvd)).
				AddMetric("latency", result.latency).
				AddLabel("ptype", "ping").
				AddLabel("probe", p.name).
				AddLabel("dst", target.Name)

			em.LatencyUnit = p.opts.LatencyUnit

			for _, al := range p.opts.AdditionalLabels {
				em.AddLabel(al.KeyValueForTarget(target.Name))
			}

			if p.opts.Validators != nil {
				em.AddMetric("validation_failure", result.validationFailure)
			}

			p.opts.LogMetrics(em)
			dataChan <- em
		}
	}
}
