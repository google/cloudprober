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

// Package message implements wrappers for sending and receiving messages with
// sequence numbers and timestamps.
package message

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
)

// FlowState maintains the state of flow on both the sender and receiver sides.
type FlowState struct {
	mu     sync.Mutex
	sender string

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
	FS        *FlowState
	Msg       *Message
	Sender    string
	Success   bool
	LostCount int
	Delayed   bool

	// Inter-packet delay = delta_recv_ts - delta_send_ts.
	// delta_send_ts = sender_ts_for_seq - sender_ts_for_seq+1 (send_ts is in msg)
	// delta_recv_ts = rcvr_ts_for_seq - rcvr_ts_for_seq+1 (ts at msg recv).
	InterPktDelay time.Duration
}

const (
	bytesInUint64 = 8
	// If msgSeq - prevSeq is lesser than this number, assume sender restart.
	delayedThreshold = 300
	// If lost count is greater than seconds in a day, assume sender restart.
	lostThreshold = 3600 * 24
)

var constants Constants

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

// NewFlowStateMap returns a new FlowStateMap variable.
func NewFlowStateMap() *FlowStateMap {
	return &FlowStateMap{
		flowState: make(map[string]*FlowState),
	}
}

// FlowState returns the flow state for the sender, creating a new one
// if necessary.
func (fm *FlowStateMap) FlowState(sender string) *FlowState {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fs, ok := fm.flowState[sender]
	if !ok {
		now := time.Now()
		fs = &FlowState{
			sender: sender,
			msgTS:  now,
			rxTS:   now,
		}
		fm.flowState[sender] = fs
	}
	return fs
}

// CreateMessage creates a message for the flow and returns byte array
// representation of the message and sequence number used on success.
func (fs *FlowState) CreateMessage(ts time.Time, maxLen int) ([]byte, uint64, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	srcNode := &DataNode{
		Name:          proto.String(fs.sender),
		TimestampUsec: Uint64ToNetworkBytes(uint64(ts.UnixNano()) / 1000),
	}
	msg := &Message{
		Magic: proto.Uint64(constants.GetMagic()),
		Seq:   Uint64ToNetworkBytes(fs.seq + 1),
		Nodes: []*DataNode{srcNode},
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

// ProcessMessage processes an incoming byte stream as a message.
// It updates FlowState for the sender seq|msgTS|rxTS and returns a
// Results object with information on the message.
func ProcessMessage(fsm *FlowStateMap, msgBytes []byte, rxTS time.Time, l *logger.Logger) (*Results, error) {
	msg := &Message{}
	if err := proto.Unmarshal(msgBytes, msg); err != nil {
		return nil, err
	}

	if len(msg.GetNodes()) == 0 {
		return nil, fmt.Errorf("invalid message: no nodes found")
	}
	srcNode := msg.GetNodes()[0]
	sender := srcNode.GetName()

	if msg.GetMagic() != constants.GetMagic() {
		return nil, fmt.Errorf("invalid message from %s: got magic %x want %x", sender, msg.GetMagic(), constants.GetMagic())
	}

	msgSeq := NetworkBytesToUint64(msg.GetSeq())
	tsVal := NetworkBytesToUint64(srcNode.GetTimestampUsec())
	msgTS := time.Unix(0, int64(tsVal)*1000)

	fs := fsm.FlowState(sender)
	fs.mu.Lock()
	defer fs.mu.Unlock()

	res := &Results{
		Msg:    msg,
		FS:     fs,
		Sender: sender,
	}

	seqDelta := int64(msgSeq - fs.seq)
	// Reset flow state if any of the conditions are met.
	// a) fs.seq == 0 => first packet in sequence.
	// b) msgSeq is too far behind => sender might have been reset.
	// c) msgSeq is too far ahead => receiver has gone out of sync for some reason.
	if fs.seq == 0 || seqDelta <= -delayedThreshold || seqDelta > lostThreshold {
		fs.seq = msgSeq
		fs.msgTS = msgTS
		fs.rxTS = rxTS
		return res, nil
	}

	if seqDelta > 0 {
		res.LostCount = int(seqDelta - 1)
		if res.LostCount == 0 {
			res.Success = true
			msgTSDelta := msgTS.Sub(fs.msgTS)
			if msgTSDelta <= 0 {
				msgTSDelta = 0
			}
			res.InterPktDelay = rxTS.Sub(fs.rxTS) - msgTSDelta
		}
		fs.seq = msgSeq
		fs.msgTS = msgTS
		fs.rxTS = rxTS
	} else if seqDelta == 0 {
		// Repeat message !!!. Prober restart? or left over message?
		l.Errorf("Duplicate seq from %s: seq %d msgTS: %s, %s", sender, msgSeq, fs.msgTS, msgTS)
	} else {
		res.Delayed = true
	}
	return res, nil
}
