package ping

import (
	"testing"
	"time"
)

func TestPreparePayload(t *testing.T) {
	ti := time.Now().UnixNano()

	timeBytes := make([]byte, 8)
	preparePayload(timeBytes, ti)

	// Verify that we send a payload that we can get the original time out of.
	for _, size := range []int{256, 1999} {
		payload := make([]byte, size)
		preparePayload(payload, ti)

		// Verify that time bytes are intact.
		ts := bytesToTime(payload)
		if ts != ti {
			t.Errorf("Got incorrect timestamp: %d, expected: %d", ts, ti)
		}
	}
}

func TestPktString(t *testing.T) {
	testPkt := &rcvdPkt{
		id:     5,
		seq:    456,
		target: "test-target",
	}
	rtt := 5 * time.Millisecond
	expectedString := "peer=test-target id=5 seq=456 rtt=5ms"
	got := testPkt.String(rtt)
	if got != expectedString {
		t.Errorf("pktString(%q, %s): expected=%s wanted=%s", testPkt, rtt, got, expectedString)
	}
}
