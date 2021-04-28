// Copyright 2017 The Cloudprober Authors.
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

package resolver

import (
	"fmt"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

type resolveBackendWithTracking struct {
	nameToIP map[string][]net.IP
	called   int
	mu       sync.Mutex
}

func (b *resolveBackendWithTracking) resolve(name string) ([]net.IP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.called++
	return b.nameToIP[name], nil
}

func (b *resolveBackendWithTracking) calls() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.called
}

func verify(testCase string, t *testing.T, ip, expectedIP net.IP, backendCalls, expectedBackendCalls int, err error) {
	if err != nil {
		t.Errorf("%s: Error while resolving. Err: %v", testCase, err)
	}
	if !ip.Equal(expectedIP) {
		t.Errorf("%s: Got wrong IP address. Got: %s, Expected: %s", testCase, ip, expectedIP)
	}
	if backendCalls != expectedBackendCalls {
		t.Errorf("%s: Backend calls: %d, Expected: %d", testCase, backendCalls, expectedBackendCalls)
	}
}

// waitForChannelOrFail reads the result from the channel and fails if it
// wasn't received within the timeout.
func waitForChannelOrFail(t *testing.T, c <-chan bool, timeout time.Duration) bool {
	select {
	case b := <-c:
		return b
	case <-time.After(timeout):
		t.Error("Channel didn't close. Stack-trace: ", debug.Stack())
		return false
	}
}

func TestResolveWithMaxAge(t *testing.T) {
	b := &resolveBackendWithTracking{
		nameToIP: make(map[string][]net.IP),
	}
	r := &Resolver{
		cache:   make(map[string]*cacheRecord),
		resolve: b.resolve,
	}

	testHost := "hostA"
	expectedIP := net.ParseIP("1.2.3.4")
	b.nameToIP[testHost] = []net.IP{expectedIP}

	// Resolve a host, there is no cache, a backend call should be made
	expectedBackendCalls := 1
	refreshed := make(chan bool, 2)
	ip, err := r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("first-run-no-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)
	// First Resolve calls refresh twice. Once for init (which succeeds), and
	// then again for refreshing, which is not needed. Hence the results are true
	// and then false.
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
	if waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned true, want false")
	}

	// Resolve same host again, it should come from cache, no backend call
	newExpectedIP := net.ParseIP("1.2.3.6")
	b.nameToIP[testHost] = []net.IP{newExpectedIP}
	ip, err = r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("second-run-from-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)
	if waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned true, want false")
	}

	// Resolve same host again with maxAge=0, it will issue an asynchronous (hence no increment
	// in expectedBackenddCalls) backend call
	ip, err = r.resolveWithMaxAge(testHost, 4, 0*time.Second, refreshed)
	verify("third-run-expire-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
	// Now that refresh has happened, we should see a new IP.
	expectedIP = newExpectedIP
	expectedBackendCalls++
	ip, err = r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("fourth-run-new-result", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)
	if waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned true, want false")
	}
}

func TestResolveErr(t *testing.T) {
	cnt := 0
	r := &Resolver{
		cache: make(map[string]*cacheRecord),
		resolve: func(name string) ([]net.IP, error) {
			cnt++
			if cnt == 2 {
				return nil, fmt.Errorf("time to return error, cnt: %d", cnt)
			}
			return []net.IP{net.ParseIP("0.0.0.0")}, nil
		},
	}
	refreshed := make(chan bool, 2)
	// cnt=0; returning 0.0.0.0.
	_, err := r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshed)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
	}
	// First Resolve calls refresh twice. Once for init (which succeeds), and
	// then again for refreshing, which is not needed. Hence the results are true
	// and then false.
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
	if waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned true, want false")
	}
	// cnt=1, returning 0.0.0.0, but updating the cache record asynchronously to contain the
	// error returned for cnt=2.
	_, err = r.resolveWithMaxAge("testHost", 4, 0*time.Second, refreshed)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
	}
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
	// cache record contains an error, and we should therefore expect an error.
	// This call for resolve will have cnt=2, and the asynchronous call to update the cache will
	// therefore update it to contain 0.0.0.0, which should be returned by the next call.
	_, err = r.resolveWithMaxAge("testHost", 4, 0*time.Second, refreshed)
	if err == nil {
		t.Errorf("Expected error, got no error")
	}
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
	// cache record now contains 0.0.0.0 again.
	_, err = r.resolveWithMaxAge("testHost", 4, 0*time.Second, refreshed)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
	}
	if !waitForChannelOrFail(t, refreshed, time.Second) {
		t.Errorf("refreshed returned false, want true")
	}
}

func TestResolveIPv6(t *testing.T) {
	b := &resolveBackendWithTracking{
		nameToIP: make(map[string][]net.IP),
	}
	r := &Resolver{
		cache:   make(map[string]*cacheRecord),
		resolve: b.resolve,
	}

	testHost := "hostA"
	expectedIPv4 := net.ParseIP("1.2.3.4")
	expectedIPv6 := net.ParseIP("::1")
	b.nameToIP[testHost] = []net.IP{expectedIPv4, expectedIPv6}

	ip, err := r.Resolve(testHost, 4)
	expectedBackendCalls := 1
	verify("ipv4-address-not-as-expected", t, ip, expectedIPv4, b.calls(), expectedBackendCalls, err)

	// This will come from cache this time, so no new backend calls.
	ip, err = r.Resolve(testHost, 6)
	verify("ipv6-address-not-as-expected", t, ip, expectedIPv6, b.calls(), expectedBackendCalls, err)

	// No IP version specified, should return IPv4 as IPv4 gets preference.
	// This will come from cache this time, so no new backend calls.
	ip, err = r.Resolve(testHost, 0)
	verify("ipv0-address-not-as-expected", t, ip, expectedIPv4, b.calls(), expectedBackendCalls, err)

	// New host, with no IPv4 address
	testHost = "hostB"
	expectedIPv6 = net.ParseIP("::2")
	b.nameToIP[testHost] = []net.IP{expectedIPv6}

	ip, err = r.Resolve(testHost, 4)
	expectedBackendCalls++
	if err == nil {
		t.Errorf("resolved IPv4 address for an IPv6 only host")
	}

	// This will come from cache this time, so no new backend calls.
	ip, err = r.Resolve(testHost, 6)
	verify("ipv6-address-not-as-expected", t, ip, expectedIPv6, b.calls(), expectedBackendCalls, err)

	// No IP version specified, should return IPv6 as there is no IPv4 address for this host.
	// This will come from cache this time, so no new backend calls.
	ip, err = r.Resolve(testHost, 0)
	verify("ipv0-address-not-as-expected", t, ip, expectedIPv6, b.calls(), expectedBackendCalls, err)
}

// TestConcurrentInit tests that multiple Resolves in parallel on the same
// target all return the same answer, and cause just 1 call to resolve.
func TestConcurrentInit(t *testing.T) {
	cnt := 0
	resolveWait := make(chan bool)
	r := &Resolver{
		cache: make(map[string]*cacheRecord),
		resolve: func(name string) ([]net.IP, error) {
			cnt++
			// The first call should be blocked on resolveWait.
			if cnt == 1 {
				<-resolveWait
				return []net.IP{net.ParseIP("0.0.0.0")}, nil
			}
			// The 2nd call should never happen.
			return nil, fmt.Errorf("resolve should be called just once, cnt: %d", cnt)
		},
	}
	// 5 because first resolve calls refresh twice.
	refreshed := make(chan bool, 5)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			_, err := r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshed)
			if err != nil {
				t.Logf("Err: %v\n", err)
				t.Errorf("Expected no error, got error")
			}
			wg.Done()
		}()
	}
	// Give offline update goroutines a chance.
	// If we call resolve more than once, this will make those resolves fail.
	runtime.Gosched()
	time.Sleep(1 * time.Millisecond)
	// Makes one of the resolve goroutines unblock refresh.
	resolveWait <- true
	resolvedCount := 0
	// 5 because first resolve calls refresh twice.
	for i := 0; i < 5; i++ {
		if waitForChannelOrFail(t, refreshed, time.Second) {
			resolvedCount++
		}
	}
	if resolvedCount != 1 {
		t.Errorf("resolvedCount=%v, want 1", resolvedCount)
	}
	wg.Wait()
}

// Set up benchmarks. Apart from performance stats it verifies the library's behavior during concurrent
// runs. It's kind of important as we use mutexes a lot, even though never in long running path, e.g.
// actual backend resolver is called outside mutexes.
//
// Use following command to run benchmark tests:
// BMC=6 BMT=4 // 6 CPUs, 4 sec
// blaze test --config=gotsan :resolver_test --test_arg=-test.bench=. \
//   --test_arg=-test.benchtime=${BMT}s --test_arg=-test.cpu=$BMC
type resolveBackendBenchmark struct {
	delay   time.Duration // artificial delay in resolving
	callCnt int64
	t       time.Time
}

func (rb *resolveBackendBenchmark) resolve(name string) ([]net.IP, error) {
	rb.callCnt++
	fmt.Printf("Time since initiation: %s\n", time.Since(rb.t))
	if rb.delay != 0 {
		time.Sleep(rb.delay)
	}
	return []net.IP{net.ParseIP("0.0.0.0")}, nil
}

func BenchmarkResolve(b *testing.B) {
	rb := &resolveBackendBenchmark{
		delay: 10 * time.Millisecond,
		t:     time.Now(),
	}
	r := &Resolver{
		cache:   make(map[string]*cacheRecord),
		resolve: rb.resolve,
	}
	// RunParallel executes its body in parallel, in multiple goroutines. Parallelism is controlled by
	// the test -cpu (test.cpu) flag (default is GOMAXPROCS). So if benchmarks runs N times, that N
	// is spread over these goroutines.
	//
	// Example benchmark results with cpu=6
	// BenchmarkResolve-6  3000	   1689466 ns/op
	//
	// 3000 is the total number of iterations (N) and it took on an average 1.69ms per iteration.
	// Total run time = 1.69 x 3000 = 5.07s. Since each goroutine executed 3000/6 or 500 iterations, with
	// each iteration taking 10ms because of artificial delay, each goroutine will take at least 5s, very
	// close to what benchmark found out.
	b.RunParallel(func(pb *testing.PB) {
		// Next() returns true if there are more iterations to execute.
		for pb.Next() {
			r.resolveWithMaxAge("test", 4, 500*time.Millisecond, nil)
			time.Sleep(10 * time.Millisecond)
		}
	})
	fmt.Printf("Called backend resolve %d times\n", rb.callCnt)
}
