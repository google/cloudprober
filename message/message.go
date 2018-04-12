// Copyright 2017-2018 Google Inc.
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

// Package message implements wrappers for sending and receiving messages with
// sequence numbers and timestamps.
package message

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	msgpb "github/google.com/cloudprober/message/proto"
)

// FlowState maintains the state of flow on both the src and dst sides.
type FlowState struct {
	mu  sync.Mutex
	src string
	dst string

	// largest sequence number received till now, except for resets.
	seq uint64
	// timestamps associated with the largest seq number message.
	msgTS time.Time
	rxTS  time.Time
}

// FlowStateMap is a container to hold all flows with a mutex to safely add new
// flows.
type FlowStateMap struct {
	mu        sync.Mutex
	flowState map[string]*FlowState
}

// Results captures the result of sequence number analysis after
// processing the latest message.
type Results struct {
	FS *FlowState

	Success   bool
	LostCount int
	Delayed   bool
	Dup       bool

	// Delta of rxTS and src timestamp in message.
	Latency time.Duration

	// Inter-packet delay for one-way-packets = delta_recv_ts - delta_send_ts.
	// delta_send_ts = sender_ts_for_seq - sender_ts_for_seq+1 (send_ts is in msg)
	// delta_recv_ts = rcvr_ts_for_seq - rcvr_ts_for_seq+1 (ts at msg recv).
	InterPktDelay time.Duration
}

const (
	bytesInUint64 = 8
	// If msgSeq - prevSeq is lesser than this number, assume src restart.
	delayedThreshold = 300
	// If lost count is greater than seconds in a day, assume src restart.
	lostThreshold = 3600 * 24
)

var constants msgpb.Constants

// Uint64ToNetworkBytes converts a 64bit unsigned integer to an 8-byte slice.
// in network byte order.
func Uint64ToNetworkBytes(val uint64) []byte {
	bytes := make([]byte, bytesInUint64)
	for i := bytesInUint64 - 1; i >= 0; i-- {
		if val <= 0 {
			break
		}
		bytes[i] = byte(val & 0xff)
		val >>= 8
	}
	return bytes
}

// NetworkBytesToUint64 converts up to first 8 bytes of input slice to a uint64.
func NetworkBytesToUint64(bytes []byte) uint64 {
	endIdx := len(bytes)
	if endIdx > bytesInUint64 {
		endIdx = bytesInUint64
	}

	val := uint64(0)
	shift := uint(0)
	for i := endIdx - 1; i >= 0; i-- {
		val += uint64(bytes[i]) << shift
		shift += 8
	}
	return val
}

// Message is a wrapper struct for the message protobuf that provides
// functions to access the most commonly accessed fields.
type Message struct {
	m *msgpb.Msg
}

// NewMessage parses a byte array into a message.
func NewMessage(msgBytes []byte) (*Message, error) {
	m := &Message{
		m: &msgpb.Msg{},
	}

	msg := m.m
	if err := proto.Unmarshal(msgBytes, msg); err != nil {
		return nil, err
	}

	if msg.GetSrc() == nil {
		return nil, fmt.Errorf("invalid message: no src node found")
	}
	if msg.GetDst() == nil {
		return nil, fmt.Errorf("invalid message: no dst node found")
	}

	if msg.GetMagic() != constants.GetMagic() {
		return nil, fmt.Errorf("invalid message from %s: got magic %x want %x", msg.GetSrc().GetName(), msg.GetMagic(), constants.GetMagic())
	}

	return m, nil
}

// Src returns the src node name.
func (m *Message) Src() string {
	return m.m.GetSrc().GetName()
}

// Dst returns the dst node name.
func (m *Message) Dst() string {
	return m.m.GetDst().GetName()
}

// Seq returns the sequence number.
func (m *Message) Seq() uint64 {
	return NetworkBytesToUint64(m.m.GetSeq())
}

// SrcTS returns the timestamp for the source.
func (m *Message) SrcTS() time.Time {
	tsVal := NetworkBytesToUint64(m.m.GetSrc().GetTimestampUsec())
	return time.Unix(0, int64(tsVal)*1000)
}

// NewFlowStateMap returns a new FlowStateMap variable.
func NewFlowStateMap() *FlowStateMap {
	return &FlowStateMap{
		flowState: make(map[string]*FlowState),
	}
}

// FlowState returns the flow state for the node, creating a new one
// if necessary.
func (fm *FlowStateMap) FlowState(src string, dst string) *FlowState {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	idx := src + dst
	fs, ok := fm.flowState[idx]
	if !ok {
		now := time.Now()
		fs = &FlowState{
			src:   src,
			dst:   dst,
			msgTS: now,
			rxTS:  now,
		}
		fm.flowState[idx] = fs
	}
	return fs
}

// SetSeq sets internal state such that the next message will contain nextSeq.
func (fs *FlowState) SetSeq(nextSeq uint64) {
	fs.seq = nextSeq - 1
}

// NextSeq returns the next sequence number that will be used.
func (fs *FlowState) NextSeq() uint64 {
	return fs.seq + 1
}

// CreateMessage creates a message for the flow and returns byte array
// representation of the message and sequence number used on success.
// TODO: add Message.CreateMessage() fn and use it in FlowState.CreateMessage.
func (fs *FlowState) CreateMessage(src string, dst string, ts time.Time, maxLen int) ([]byte, uint64, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dstType := msgpb.DataNode_SERVER
	msg := &msgpb.Msg{
		Magic: proto.Uint64(constants.GetMagic()),
		Seq:   Uint64ToNetworkBytes(fs.seq + 1),
		Src: &msgpb.DataNode{
			Name:          proto.String(src),
			TimestampUsec: Uint64ToNetworkBytes(uint64(ts.UnixNano()) / 1000),
		},
		Dst: &msgpb.DataNode{
			Name:          proto.String(dst),
			TimestampUsec: Uint64ToNetworkBytes(uint64(0)),
			Type:          &dstType,
		},
	}
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, 0, err
	}

	if len(bytes) > maxLen {
		return nil, 0, fmt.Errorf("marshalled message too long %d > allowed max %d", len(bytes), maxLen)
	}
	fs.seq++
	return bytes, fs.seq, nil
}

// WithdrawMessage tries to update internal state that message with "seq"
// was not sent. If no new packet was sent in parallel, then sequence number
// is decremented, and the function returns true. Otherwise, it is a no-op
// and the function returns false.
func (fs *FlowState) WithdrawMessage(seq uint64) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.seq == seq {
		fs.seq--
		return true
	}
	return false
}

// ProcessOneWay processes a one-way message on the receiving end.
// It updates FlowState for the sender seq|msgTS|rxTS and returns a
// Results object with metrics derived from the message.
func (m *Message) ProcessOneWay(fsm *FlowStateMap, rxTS time.Time) *Results {
	srcTS := m.SrcTS()
	res := &Results{
		Latency: rxTS.Sub(srcTS),
	}

	fs := fsm.FlowState(m.Src(), m.Dst())
	fs.mu.Lock()
	defer fs.mu.Unlock()
	res.FS = fs

	msgSeq := m.Seq()
	seqDelta := int64(msgSeq - fs.seq)
	// Reset flow state if any of the conditions are met.
	// a) fs.seq == 0 => first packet in sequence.
	// b) m.Seq is too far behind => src might have been reset.
	// c) m.Seq is too far ahead => receiver has gone out of sync for some reason.
	if fs.seq == 0 || seqDelta <= -delayedThreshold || seqDelta > lostThreshold {
		fs.seq = msgSeq
		fs.msgTS = srcTS
		fs.rxTS = rxTS
		return res
	}

	if seqDelta > 0 {
		res.LostCount = int(seqDelta - 1)
		if res.LostCount == 0 {
			res.Success = true
			msgTSDelta := srcTS.Sub(fs.msgTS)
			if msgTSDelta <= 0 {
				msgTSDelta = 0
			}
			res.InterPktDelay = rxTS.Sub(fs.rxTS) - msgTSDelta
		}
		fs.seq = msgSeq
		fs.msgTS = srcTS
		fs.rxTS = rxTS
	} else if seqDelta == 0 {
		// Repeat message !!!. Prober restart? or left over message?
		res.Dup = true
	} else {
		res.Delayed = true
	}
	return res
}
