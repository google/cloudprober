package ping

import (
	"math/rand"
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

		// Verify payload.
		err := verifyPayload(bytesBuf)
		if err != nil {
			t.Errorf("Got bad data from timeToBytes: %v", err)
		}

		// Verify that time bytes are intact.
		ts := bytesToTime(bytesBuf)
		if ts.UnixNano() != ti.UnixNano() {
			t.Errorf("Got incorrect timestamp: %d, expected: %d", ts.UnixNano(), ti.UnixNano())
		}

		// Verify corruption results in failure.
		bytesBuf[rand.Intn(len(bytesBuf))] = byte('a')
		if verifyPayload(bytesBuf) == nil {
			t.Error("Expected data integrity check error, but got none.")
		}
	}
}

func benchmarkVerifyPayload(size int, b *testing.B) {
	ts := time.Now()
	bytesBuf := timeToBytes(ts, size)
	for n := 0; n < b.N; n++ {
		verifyPayload(bytesBuf)
	}
}

func BenchmarkVerifyPayload56(b *testing.B)   { benchmarkVerifyPayload(56, b) }
func BenchmarkVerifyPayload256(b *testing.B)  { benchmarkVerifyPayload(256, b) }
func BenchmarkVerifyPayload1999(b *testing.B) { benchmarkVerifyPayload(1999, b) }
func BenchmarkVerifyPayload9999(b *testing.B) { benchmarkVerifyPayload(9999, b) }
