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

// Package resolver provides a caching, non-blocking DNS resolver. All requests
// for cached resources are returned immediately and if cache has expired, an
// offline goroutine is fired to update it.
package resolver

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// The max age and the timeout for resolving a target.
const defaultMaxAge = 5 * time.Minute

type cacheRecord struct {
	ip4              net.IP
	ip6              net.IP
	lastUpdatedAt    time.Time
	err              error
	mu               sync.Mutex
	updateInProgress bool
	callInit         sync.Once
}

// Resolver provides an asynchronous caching DNS resolver.
type Resolver struct {
	cache         map[string]*cacheRecord
	mu            sync.Mutex
	DefaultMaxAge time.Duration
	resolve       func(string) ([]net.IP, error) // used for testing
}

// ipVersion tells if an IP address is IPv4 or IPv6.
func ipVersion(ip net.IP) int {
	if len(ip.To4()) == net.IPv4len {
		return 4
	}
	if len(ip) == net.IPv6len {
		return 6
	}
	return 0
}

// resolveOrTimeout tries to resolve, but times out and returns an error if it
// takes more than defaultMaxAge.
// Has the potential of creating a bunch of pending goroutines if backend
// resolve call has a tendency of indefinitely hanging.
func (r *Resolver) resolveOrTimeout(name string) ([]net.IP, error) {
	var ips []net.IP
	var err error
	doneChan := make(chan struct{})

	go func() {
		ips, err = r.resolve(name)
		close(doneChan)
	}()

	select {
	case <-doneChan:
		return ips, err
	case <-time.After(defaultMaxAge):
		return nil, fmt.Errorf("timed out after %v", defaultMaxAge)
	}
}

// Resolve returns IP address for a name.
// Issues an update call for the cache record if it's older than defaultMaxAge.
func (r *Resolver) Resolve(name string, ipVer int) (net.IP, error) {
	maxAge := r.DefaultMaxAge
	if maxAge == 0 {
		maxAge = defaultMaxAge
	}
	return r.resolveWithMaxAge(name, ipVer, maxAge, nil)
}

// getCacheRecord returns the cache record for the target.
// It must be kept light, as it blocks the main mutex of the map.
func (r *Resolver) getCacheRecord(name string) *cacheRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	cr := r.cache[name]
	if cr == nil {
		cr = &cacheRecord{
			err: errors.New("cache record not initialized yet"),
		}
		r.cache[name] = cr
	}
	return cr
}

// resolveWithMaxAge returns IP address for a name, issuing an update call for
// the cache record if it's older than the argument maxAge.
// refreshed channel is primarily used for testing. Method pushes true to
// refreshed channel once and if the value is refreshed, or false, if it
// doesn't need refreshing.
func (r *Resolver) resolveWithMaxAge(name string, ipVer int, maxAge time.Duration, refreshed chan<- bool) (net.IP, error) {
	cr := r.getCacheRecord(name)
	cr.refreshIfRequired(name, r.resolveOrTimeout, maxAge, refreshed)
	cr.mu.Lock()
	defer cr.mu.Unlock()
	ip := cr.ip4
	if ipVer == 6 {
		ip = cr.ip6
	}
	if ip == nil && cr.err == nil {
		return nil, fmt.Errorf("found no IP%d IP for %s", ipVer, name)
	}
	return ip, cr.err
}

// refresh refreshes the cacheRecord by making a call to the provided "resolve" function.
func (cr *cacheRecord) refresh(name string, resolve func(string) ([]net.IP, error), refreshed chan<- bool) {
	// Note that we call backend's resolve outside of the mutex locks and take the lock again
	// to update the cache record once we have the results from the backend.
	ips, err := resolve(name)

	cr.mu.Lock()
	defer cr.mu.Unlock()
	if refreshed != nil {
		refreshed <- true
	}
	cr.err = err
	cr.lastUpdatedAt = time.Now()
	cr.updateInProgress = false
	if err != nil {
		return
	}
	cr.ip4 = nil
	cr.ip6 = nil
	for _, ip := range ips {
		switch ipVersion(ip) {
		case 4:
			cr.ip4 = ip
		case 6:
			cr.ip6 = ip
		}
	}
}

// refreshIfRequired does most of the work. Overall goal is to minimize the
// lock period of the cache record. To that end, if the cache record needs
// updating, we do that with the mutex unlocked.
//
// If cache record is new, blocks until it's resolved for the first time.
// If cache record needs updating, kicks off refresh asynchronously.
// If cache record is already being updated or fresh enough, returns immediately.
func (cr *cacheRecord) refreshIfRequired(name string, resolve func(string) ([]net.IP, error), maxAge time.Duration, refreshed chan<- bool) {
	cr.callInit.Do(func() { cr.refresh(name, resolve, refreshed) })
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Cache record is old and no update in progress, issue a request to update.
	if !cr.updateInProgress && time.Since(cr.lastUpdatedAt) >= maxAge {
		cr.updateInProgress = true
		go cr.refresh(name, resolve, refreshed)
	} else if refreshed != nil {
		refreshed <- false
	}
}

// New returns a new Resolver.
func New() *Resolver {
	return &Resolver{
		cache:         make(map[string]*cacheRecord),
		resolve:       func(name string) ([]net.IP, error) { return net.LookupIP(name) },
		DefaultMaxAge: defaultMaxAge,
	}
}
