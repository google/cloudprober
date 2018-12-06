package ping

import (
	"reflect"
	"testing"
	"time"
)

func TestTimeToBytes(t *testing.T) {
	ti := time.Now()

	timeBytes := timeToBytes(ti, 8)

	// Verify that for the larger payload sizes we get replicas of the same
	// bytes.
	for _, size := range []int{256, 1999} {
		bytesBuf := timeToBytes(ti, size)

		var expectedBuf []byte
		for i := 0; i < size/8; i++ {
			expectedBuf = append(expectedBuf, timeBytes...)
		}
		// Pad 0s in the end.
		expectedBuf = append(expectedBuf, make([]byte, size-len(expectedBuf))...)
		if !reflect.DeepEqual(bytesBuf, expectedBuf) {
			t.Errorf("Bytes array:\n%v\n\nExpected:\n%v", bytesBuf, expectedBuf)
		}

		// Verify that time bytes are intact.
		ts := bytesToTime(bytesBuf)
		if ts.UnixNano() != ti.UnixNano() {
			t.Errorf("Got incorrect timestamp: %d, expected: %d", ts.UnixNano(), ti.UnixNano())
		}
	}
}
