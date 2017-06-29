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

const defaultMaxAge = 5 * time.Minute

type cacheRecord struct {
	ip4              net.IP
	ip6              net.IP
	lastUpdatedAt    time.Time
	err              error
	mu               sync.Mutex
	updateInProgress bool
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

// Resolve returns IP address for a name. It's same as ResolveWithMaxAge, except for that it uses
// DefaultMaxAge for validating the cache record.
func (r *Resolver) Resolve(name string, ipVer int) (net.IP, error) {
	maxAge := r.DefaultMaxAge
	if maxAge == 0 {
		maxAge = defaultMaxAge
	}
	return r.ResolveWithMaxAge(name, ipVer, maxAge)
}

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

// ResolveWithMaxAge returns IP address for a name, issuing an update call for the cache record if
// it's older than the argument maxAge.
func (r *Resolver) ResolveWithMaxAge(name string, ipVer int, maxAge time.Duration) (net.IP, error) {
	cr := r.getCacheRecord(name)
	cr.refreshIfRequired(name, r.resolve, maxAge)
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

// refreshIfRequired does most of the work. Overall goal is to minimize the lock period of the cache record. To that
// end, if the cache record needs updating, we do that with the mutex unlocked.
//
// If cache record is already being updated or fresh enough it returns immediately.
// if cache record needs updating, we kick off refresh in a new goroutine. If this is not the first update, we immediately
// return whatever is in the cache. However, if it's the first update, we wait for the refresh to finish, before returning.
//
// TODO: See if this whole magic will be simpler to reason-about if we use channels.
func (cr *cacheRecord) refreshIfRequired(name string, resolve func(string) ([]net.IP, error), maxAge time.Duration) {
	cr.mu.Lock()

	// If no need to refresh or update in progress
	if cr.updateInProgress || time.Since(cr.lastUpdatedAt) < maxAge {
		cr.mu.Unlock()
		return
	}

	// Cache record is old and no update in progress, issue a request to update.
	cr.updateInProgress = true
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Note that we call backend's resolve outside of the mutex locks and take the lock again
		// to update the cache record once we have the results from the backend.
		ips, err := resolve(name)
		cr.mu.Lock()
		defer cr.mu.Unlock()
		cr.err = err
		if err != nil {
			// Reset cache record time so it's updated inline next time.
			cr.lastUpdatedAt = time.Time{}
			cr.updateInProgress = false
			return
		}
		// Reset error state
		cr.err = nil
		var ip4, ip6 net.IP
		for _, ip := range ips {
			switch ipVersion(ip) {
			case 4:
				ip4 = ip
			case 6:
				ip6 = ip
			}
		}
		cr.ip4 = ip4
		cr.ip6 = ip6
		cr.lastUpdatedAt = time.Now()
		cr.updateInProgress = false
	}()

	// Wait if this is first update
	if cr.lastUpdatedAt.IsZero() {
		cr.mu.Unlock()
		wg.Wait()
	} else {
		cr.mu.Unlock()
	}
	return
}

// New returns a new Resolver.
func New() *Resolver {
	return &Resolver{
		cache:         make(map[string]*cacheRecord),
		resolve:       func(name string) ([]net.IP, error) { return net.LookupIP(name) },
		DefaultMaxAge: defaultMaxAge,
	}
}
