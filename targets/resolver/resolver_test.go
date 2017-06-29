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

package resolver

import (
	"fmt"
	"net"
	"runtime"
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
	ip, err := r.ResolveWithMaxAge(testHost, 4, 60*time.Second)
	verify("first-run-no-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)

	// Resolve same host again, it should come from cache, no backend call
	newExpectedIP := net.ParseIP("1.2.3.6")
	b.nameToIP[testHost] = []net.IP{newExpectedIP}
	ip, err = r.ResolveWithMaxAge(testHost, 4, 60*time.Second)
	verify("second-run-from-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)

	// Resolve same host again with maxAge=0, it will issue an asynchronous (hence no increment
	// in expectedBackenddCalls) backend call
	ip, err = r.ResolveWithMaxAge(testHost, 4, 0*time.Second)
	verify("third-run-expire-cache", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)

	// Give other goroutines a chance.
	runtime.Gosched()
	time.Sleep(1 * time.Millisecond)
	// Resolve same host again, we should see new IP now.
	expectedIP = newExpectedIP
	expectedBackendCalls++
	ip, err = r.ResolveWithMaxAge(testHost, 4, 60*time.Second)
	verify("fourth-run-new-result", t, ip, expectedIP, b.calls(), expectedBackendCalls, err)
}

func TestResolveErr(t *testing.T) {
	cnt := 0
	r := &Resolver{
		cache: make(map[string]*cacheRecord),
		resolve: func(name string) ([]net.IP, error) {
			cnt++
			if cnt == 2 || cnt == 3 {
				return nil, fmt.Errorf("time to return error, cnt: %d", cnt)
			}
			return []net.IP{net.ParseIP("0.0.0.0")}, nil
		},
	}
	// Backend resolve called, cnt = 1
	_, err := r.ResolveWithMaxAge("testHost", 4, 0*time.Second)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
	}
	// Backend resolve not called yet, old result, cnt = 1
	_, err = r.ResolveWithMaxAge("testHost", 4, 0*time.Second)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
	}
	// Give offline update goroutines a chance.
	runtime.Gosched()
	time.Sleep(1 * time.Millisecond)

	// Backend's resolve has been called (with cnt = 2), cr.err is set right now.
	// An error resets the cache record, causing next resolve call to reach out to
	// backend's resolver again and wait for it before returning.
	// However backend resolver will again return an error as cnt=3.
	_, err = r.ResolveWithMaxAge("testHost", 4, 0*time.Second)
	if err == nil {
		t.Errorf("Expected error, got no error")
	}
	_, err = r.ResolveWithMaxAge("testHost", 4, 0*time.Second)
	if err != nil {
		t.Logf("Err: %v\n", err)
		t.Errorf("Expected no error, got error")
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
			r.ResolveWithMaxAge("test", 4, 500*time.Millisecond)
			time.Sleep(10 * time.Millisecond)
		}
	})
	fmt.Printf("Called backend resolve %d times\n", rb.callCnt)
}
