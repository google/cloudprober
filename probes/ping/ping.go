// Copyright 2017 Google Inc.
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
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/ping/proto"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/validators"
	"github.com/google/cloudprober/validators/integrity"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	protocolICMP     = 1
	protocolIPv6ICMP = 58
	dataIntegrityKey = "data-integrity"
)

// Probe implements a ping probe type that sends ICMP ping packets to the targets and reports
// back statistics on packets sent, received and the rtt.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	source            string
	ipVer             int
	targets           []string
	sent              map[string]int64
	received          map[string]int64
	latency           map[string]metrics.Value
	validationFailure map[string]*metrics.Map
	conn              icmpConn
	runCnt            uint64
	target2addr       map[string]net.Addr
	ip2target         map[string]string
}

// Init initliazes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("no ping config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	if p.c.GetPayloadSize() < 8 {
		return fmt.Errorf("payload_size (%d) cannot be smaller than 8", p.c.GetPayloadSize())
	}
	p.ipVer = int(p.c.GetIpVersion())
	p.sent = make(map[string]int64)
	p.received = make(map[string]int64)
	p.latency = make(map[string]metrics.Value)
	p.validationFailure = make(map[string]*metrics.Map)
	p.ip2target = make(map[string]string)
	p.target2addr = make(map[string]net.Addr)

	if err := p.configureIntegrityCheck(); err != nil {
		return err
	}

	if err := p.setSourceFromConfig(); err != nil {
		return err
	}
	return p.listen()
}

func (p *Probe) latencyForTarget(target string) metrics.Value {
	if val, ok := p.latency[target]; ok {
		return val
	}

	var latencyValue metrics.Value
	if p.opts.LatencyDist != nil {
		latencyValue = p.opts.LatencyDist.Clone()
	} else {
		latencyValue = metrics.NewFloat(0)
	}
	p.latency[target] = latencyValue

	return latencyValue
}

// Adds an integrity validator if data integrity checks are not disabled.
func (p *Probe) configureIntegrityCheck() error {
	if p.c.GetDisableIntegrityCheck() {
		return nil
	}

	if p.opts.Validators["data-integrity"] != nil {
		p.l.Warningf("Not adding data-integrity validator as there is already a validator with the name \"data-integrity\": %v", p.opts.Validators["data-integrity"])
		return nil
	}

	v, err := integrity.PatternNumBytesValidator(timeBytesSize, p.l)
	if err != nil {
		return err
	}

	if p.opts.Validators == nil {
		p.opts.Validators = make(map[string]validators.Validator)
	}
	p.opts.Validators[dataIntegrityKey] = v

	return nil
}

// setSourceFromConfig sets the source for ping probes. This is where we listen
// for the replies.
func (p *Probe) setSourceFromConfig() error {
	switch p.c.Source.(type) {
	case *configpb.ProbeConf_SourceIp:
		p.source = p.c.GetSourceIp()
	case *configpb.ProbeConf_SourceInterface:
		s, err := probeutils.ResolveIntfAddr(p.c.GetSourceInterface())
		if err != nil {
			return err
		}
		p.l.Infof("Using %v as source address.", p.source)
		p.source = s
	default:
		p.source = ""
	}
	return nil
}

func (p *Probe) listen() error {
	netProto := "ip4:icmp"
	if p.ipVer == 6 {
		netProto = "ip6:ipv6-icmp"
	}

	if p.c.GetUseDatagramSocket() {
		// udp network represents datagram ICMP sockets. The name is a bit
		// misleading, but that's what Go's icmp package uses.
		netProto = fmt.Sprintf("udp%d", p.ipVer)
	}

	// If source is configured, try to resolve it to make sure it has the same
	// IP version as p.ipVer.
	if p.source != "" {
		sourceIP, err := resolveAddr(p.source, p.ipVer)
		if err != nil || sourceIP == nil {
			return fmt.Errorf("Bad source address: %s, Err: %v", p.source, err)
		}
	}

	var err error
	p.conn, err = newICMPConn(netProto, p.source)
	return err
}

func (p *Probe) resolveTargets() {
	for _, t := range p.targets {
		ip, err := p.opts.Targets.Resolve(t, p.ipVer)
		if err != nil {
			p.l.Warningf("Bad target: %s. Err: %v", t, err)
			p.target2addr[t] = nil
			continue
		}
		p.l.Debugf("target: %s, resolved ip: %v", t, ip)
		var a net.Addr
		a = &net.IPAddr{IP: ip}
		if p.c.GetUseDatagramSocket() {
			a = &net.UDPAddr{IP: ip}
		}
		p.target2addr[t] = a
		p.ip2target[ip.String()] = t
	}
}

// Match first 8-bits of the run ID with the first 8-bits of the ICMP sequence number.
// For raw sockets we also match ICMP id with the run ID. For datagram sockets, kernel
// does that matching for us. It rewrites the ICMP id of the outgoing packets with the
// fake local port selected at the time of the socket creation and uses the same criteria
// to forward incoming packets to the sockets.
func matchPacket(runID uint16, pktID, pktSeq int, datagramSocket bool) bool {
	return runID>>8 == uint16(pktSeq)>>8 && (datagramSocket || uint16(pktID) == runID)
}

func (p *Probe) packetToSend(runID, seq uint16) []byte {
	var typ icmp.Type
	typ = ipv4.ICMPTypeEcho
	if p.ipVer == 6 {
		typ = ipv6.ICMPTypeEchoRequest
	}
	pbytes, err := (&icmp.Message{
		Type: typ, Code: 0,
		Body: &icmp.Echo{
			ID: int(runID), Seq: int(seq),
			Data: timeToBytes(time.Now(), int(p.c.GetPayloadSize())),
		},
	}).Marshal(nil)
	if err != nil {
		// It should never happen.
		p.l.Criticalf("Error marshalling the ICMP message. Err: %v", err)
	}
	return pbytes
}

func (p *Probe) sendPackets(runID uint16, tracker chan bool) {
	seq := runID & uint16(0xff00)
	for i := int32(0); i < p.c.GetPacketsPerProbe(); i++ {
		for _, target := range p.targets {
			if p.target2addr[target] == nil {
				p.l.Debugf("Skipping unresolved target: %s", target)
				continue
			}
			p.l.Debugf("Request to=%s id=%d seq=%d", target, runID, seq)
			if _, err := p.conn.write(p.packetToSend(runID, seq), p.target2addr[target]); err != nil {
				p.l.Warning(err.Error())
				continue
			}
			tracker <- true
			p.sent[target]++
		}
		seq++
		time.Sleep(time.Duration(p.c.GetPacketsIntervalMsec()) * time.Millisecond)
	}
	p.l.Debugf("%s: Done sending packets, closing the tracker.", p.name)
	close(tracker)
}

func (p *Probe) fetchPacket(pktbuf []byte) (net.IP, *icmp.Message, error) {
	n, peer, err := p.conn.read(pktbuf)
	if err != nil {
		return nil, nil, err
	}
	var peerIP net.IP
	switch peer := peer.(type) {
	case *net.UDPAddr:
		peerIP = peer.IP
	case *net.IPAddr:
		peerIP = peer.IP
	}
	proto := protocolICMP
	if p.ipVer == 6 {
		proto = protocolIPv6ICMP
	}
	m, err := icmp.ParseMessage(proto, pktbuf[:n])
	if err != nil {
		return nil, nil, err
	}
	return peerIP, m, nil
}

func (p *Probe) recvPackets(runID uint16, tracker chan bool) {
	received := make(map[string]bool)
	outstandingPkts := 0
	p.conn.setReadDeadline(time.Now().Add(p.opts.Timeout))
	pktbuf := make([]byte, 1500)
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
		peerIP, m, err := p.fetchPacket(pktbuf)
		if err != nil {
			p.l.Warning(err.Error())
			if neterr, ok := err.(*net.OpError); ok && neterr.Timeout() {
				return
			}
		}
		target := p.ip2target[peerIP.String()]
		if target == "" {
			p.l.Debugf("Got a packet from a peer that's not one of my targets: %s\n", peerIP.String())
			continue
		}
		if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
			continue
		}

		pkt, ok := m.Body.(*icmp.Echo)
		if !ok {
			p.l.Error("Got wrong packet in ICMP echo reply.") // should never happen
			continue
		}

		rtt := time.Since(bytesToTime(pkt.Data))

		// check if this packet belongs to this run
		if !matchPacket(runID, pkt.ID, pkt.Seq, p.c.GetUseDatagramSocket()) {
			p.l.Infof("Reply from=%s id=%d seq=%d rtt=%s Unmatched packet, probably from the last probe run.", target, pkt.ID, pkt.Seq, rtt)
			continue
		}

		key := fmt.Sprintf("%s_%d", target, pkt.Seq)
		// Check if we have already seen this packet.
		if received[key] {
			p.l.Infof("Duplicate reply from=%s id=%d seq=%d rtt=%s (DUP)", target, pkt.ID, pkt.Seq, rtt)
			continue
		}
		received[key] = true

		// Read a "good" packet, where good means that it's one of the replies that
		// we were looking for.
		outstandingPkts--

		if p.opts.Validators != nil {
			var failedValidations []string

			for name, v := range p.opts.Validators {
				success, err := v.Validate(nil, pkt.Data)
				if err != nil {
					p.l.Errorf("Error while running the validator %s: %v", name, err)
					continue
				}

				if !success {
					if p.validationFailure[target] == nil {
						p.validationFailure[target] = metrics.NewMap("validator", &metrics.Int{})
					}
					p.validationFailure[target].IncKey(name)
					failedValidations = append(failedValidations, name)
				}
			}

			// If any validation failed, return now, leaving the success and latency counters unchanged.
			if len(failedValidations) > 0 {
				p.l.Debugf("Target:%s, ping.recvPackets: failed validations: %v.", target, failedValidations)
				continue
			}
		}

		p.received[target]++
		p.latencyForTarget(target).AddFloat64(rtt.Seconds() / p.opts.LatencyUnit.Seconds())
		p.l.Debugf("Reply from=%s id=%d seq=%d rtt=%s", target, pkt.ID, pkt.Seq, rtt)
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
		p.targets = p.opts.Targets.List()
		p.resolveTargets()
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
	for ts := range time.Tick(p.opts.Interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.runProbe()
		p.l.Debugf("%s: Probe finished.", p.name)
		if (p.runCnt % uint64(p.c.GetStatsExportInterval())) != 0 {
			continue
		}
		for _, t := range p.targets {
			em := metrics.NewEventMetrics(ts).
				AddMetric("total", metrics.NewInt(p.sent[t])).
				AddMetric("success", metrics.NewInt(p.received[t])).
				AddMetric("latency", p.latencyForTarget(t)).
				AddLabel("ptype", "ping").
				AddLabel("probe", p.name).
				AddLabel("dst", t)

			dataChan <- em
			p.l.Info(em.String())
		}
	}
}
