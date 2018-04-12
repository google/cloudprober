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

package message

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	msgpb "github.com/google/cloudprober/message/proto"
)

// createMessage is a helper function for creating a message and fatally failing
// if CreateMessage fails. This is for use in places where we don't expect
// CreateMessage to fail.
func createMessage(t *testing.T, src, dst string, fs *FlowState, ts time.Time) ([]byte, uint64) {
	maxLen := 1024
	msgBytes, msgSeq, err := fs.CreateMessage(src, dst, ts, maxLen)
	if err != nil {
		t.Fatalf("Error creating message for seq %d: %v", fs.seq+1, err)
	}
	return msgBytes, msgSeq
}

// TestUint64Conversion tests the conversion of uint64 from and to network byte order.
func TestUint64Conversion(t *testing.T) {
	val := uint64(0)
	for i := uint64(0); i < 10; i++ {
		inp := val + i
		bytes := Uint64ToNetworkBytes(val + i)
		out := NetworkBytesToUint64(bytes)
		if inp != out {
			t.Errorf("Conversion pipeline failed: got %d want %d", out, inp)
		}
		inp = val - i
		bytes = Uint64ToNetworkBytes(inp)
		out = NetworkBytesToUint64(bytes)
		if inp != out {
			t.Errorf("Conversion pipeline failed: got %d want %d", out, inp)
		}
	}
}

// TestMessageEncodeDecode tests encoding/decoding of properly formed msgs.
func TestMessageEncodeDecode(t *testing.T) {
	txFSM := NewFlowStateMap()
	rxFSM := NewFlowStateMap()

	src := "aa-src"
	dst := "zz-dst"
	seq := uint64(100)
	ts := time.Now().Truncate(time.Microsecond)

	txFS := txFSM.FlowState(src, dst)
	txFS.seq = seq - 1
	msgBytes, msgSeq := createMessage(t, src, dst, txFS, ts)
	if msgSeq != seq {
		t.Errorf("Incorrect seq in message: got %d want %d", msgSeq, seq)
	}

	// Pre-create flow state on the rx side. So we expect success in every step.
	rxFS := rxFSM.FlowState(src, dst)
	rxFS.seq = seq - 1
	msg, err := NewMessage(msgBytes)
	if err != nil {
		t.Fatalf("Process message failure: %v", err)
	}
	if msg.Src() != src || msg.Dst() != dst {
		t.Errorf("Message content error (src, dst): got (%s, %s) want (%s, %s)", msg.Src(), msg.Dst(), src, dst)
	}
	res := msg.ProcessOneWay(rxFSM, ts.Add(time.Second))
	if rxFS.seq != seq {
		t.Errorf("Seq number mismatch. got %d want %d. %v %v", rxFS.seq, seq, rxFS, res)
	}
	if !res.Success || res.LostCount > 0 || res.Delayed {
		t.Errorf("Success, lostCount, delayed mismatch. got (%v %v %v) want (%v %v %v)",
			res.Success, res.LostCount, res.Delayed, true, 0, false)
	}
}

// TestInvalidMessages tests encoding/decoding error paths.
func TestInvalidMessage(t *testing.T) {
	fss := NewFlowStateMap()

	src := "aa-src"
	dst := "zz-dst"
	seq := uint64(100)
	ts := time.Now().Truncate(time.Microsecond)
	maxLen := 10

	fs := fss.FlowState(src, dst)
	if msgBytes, _, err := fs.CreateMessage(src, dst, ts, maxLen); err == nil {
		t.Errorf("Message too long(%d) for maxlen(%d) but did not fail.", len(msgBytes), maxLen)
	}

	// Invalid magic.
	msg := &msgpb.Msg{
		Magic: proto.Uint64(constants.GetMagic() + 1),
		Seq:   Uint64ToNetworkBytes(seq),
		Src: &msgpb.DataNode{
			Name: proto.String(src),
		},
		Dst: &msgpb.DataNode{
			Name: proto.String(dst),
		},
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Error marshalling message: %v", err)
	}

	if _, err := NewMessage(msgBytes); err == nil {
		t.Error("ProcessMessage expected to fail due to invalid magic but did not fail")
	}
}

// TestSeqHandling tests various sequence number cases.
func TestSeqHandling(t *testing.T) {
	txFSM := NewFlowStateMap()
	rxFSM := NewFlowStateMap()

	src := "aa-src"
	dst := "zz-dst"
	seq := uint64(100)
	pktTS := time.Now().Truncate(time.Microsecond)
	rxTS := pktTS.Add(time.Millisecond)

	// Create a message and revert it.
	txFS := txFSM.FlowState(src, dst)
	txFS.seq = seq - 1
	msgBytes, msgSeq := createMessage(t, src, dst, txFS, pktTS)

	if !txFS.WithdrawMessage(msgSeq) || txFS.seq != seq-1 {
		t.Errorf("WithdrawMessage failed: NextSeq %d msgSeq %d fs.seq %d", seq, msgSeq, txFS.seq)
	}
	// withdraw an older message and expect failure.
	if txFS.WithdrawMessage(seq - 2) {
		t.Errorf("WithdrawMessage succeeded: msgSeq %d fs.seq %d", seq-2, txFS.seq)
	}

	txFS.seq = seq - 1
	msgBytes, msgSeq = createMessage(t, src, dst, txFS, pktTS)
	// Receive the message and process it. Seq and srcs should match.
	// This will be the first message for the flow:
	//		=> Flowstate should be created.
	// 		=> We do not expect success in the result.
	msg, err := NewMessage(msgBytes)
	if err != nil {
		t.Fatalf("Error processing message: %v", err)
	}
	res := msg.ProcessOneWay(rxFSM, rxTS)
	rxFS := res.FS
	if rxFS == nil || rxFSM.FlowState(src, dst) != rxFS {
		t.Errorf("Expected sender to appear in FlowStateMap struct, got %v", rxFSM.FlowState(src, dst))
	}
	if rxFS.src != src {
		t.Errorf("Message content error - src: got %s want %s", rxFS.src, src)
	}
	if rxFS.seq != seq {
		t.Errorf("Seq number mismatch. got %d want %d.", rxFS.seq, seq)
	}
	if res.Success || res.LostCount > 0 || res.Delayed {
		t.Errorf("Success, lostCount, delayed mismatch. got (%v %v %v) want (%v %v %v)",
			res.Success, res.LostCount, res.Delayed, false, 0, false)
	}

	// Send a message with an older seq number.
	pktTS = pktTS.Add(time.Second)
	rxTS = rxTS.Add(time.Second)
	txFS.seq = seq - 10
	msgBytes, msgSeq = createMessage(t, src, dst, txFS, pktTS)
	if res.FS.msgTS == pktTS || res.FS.rxTS == rxTS {
		t.Errorf("Timestamps not updated. got (%s, %s) want (%s, %s)", res.FS.msgTS, res.FS.rxTS, pktTS, rxTS)
	}

	// Send a message with seq+1.
	// 		pktTS = prevPktTS + 1sec.
	// 		rxTS = prevRxTS + 1 sec + 1 millisecond.
	// 		=> ipd = 1 millisecond
	ipd := time.Millisecond
	pktTS = pktTS.Add(time.Second)
	rxTS = rxTS.Add(time.Second + ipd)
	txFS.seq = seq
	msgBytes, msgSeq = createMessage(t, src, dst, txFS, pktTS)
	if msg, err = NewMessage(msgBytes); err != nil {
		t.Fatalf("Error processing message: %v", err)
	}
	res = msg.ProcessOneWay(rxFSM, rxTS)
	rxFS = res.FS
	if !res.Success {
		t.Errorf("Got failure, want success. tx seq: %d, rx want seq: %d", txFS.seq, rxFS.seq+1)
	}
	if res.InterPktDelay != ipd {
		t.Errorf("InterPktDelay calculation error got %v want %v", res.InterPktDelay, ipd)
	}
	if res.FS.msgTS != pktTS || res.FS.rxTS != rxTS {
		t.Errorf("Timestamps not updated. got (%s, %s) want (%s, %s)", res.FS.msgTS, res.FS.rxTS, pktTS, rxTS)
	}

	// Send a message with lost packets = 10.
	lost := 10
	txFS.seq += uint64(lost)
	pktTS = pktTS.Add(time.Second)
	rxTS = rxTS.Add(time.Second)
	msgBytes, msgSeq = createMessage(t, src, dst, txFS, pktTS)
	if msg, err = NewMessage(msgBytes); err != nil {
		t.Fatalf("Error processing message: %v", err)
	}
	res = msg.ProcessOneWay(rxFSM, rxTS)
	rxFS = res.FS
	if res.Success {
		t.Error("Got success, want failure.")
	}
	if res.LostCount != lost {
		t.Errorf("Got lostcount=%d want %d tx seq: %d, rx want seq: %d", res.LostCount, lost, txFS.seq, rxFS.seq+1)
	}
	if res.FS.msgTS != pktTS || res.FS.rxTS != rxTS {
		t.Errorf("Timestamps not updated. got (%s, %s) want (%s, %s)", res.FS.msgTS, res.FS.rxTS, pktTS, rxTS)
	}
}
