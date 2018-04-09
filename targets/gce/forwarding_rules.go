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

package gce

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// globalForwardingRules is a singleton instance of the forwardingRules struct.
// It is presented as a singleton because, like instances, forwardingRules provides
// a cache layer that is best shared by all probes.
var (
	globalForwardingRules *forwardingRules
	onceForwardingRules   sync.Once
)

// forwardingRules is a lister which lists GCE forwarding rules (see
// https://cloud.google.com/compute/docs/load-balancing/network/forwarding-rules
// for information on forwarding rules). In addition to being able to list the
// rules, this particular lister implements a cache. On a timer (configured by
// GlobalGCETargetsOptions.re_eval_sec cloudprober/targets/targets.proto) in the
// background the cache will be populated (by the equivalent of running "gcloud
// compute forwarding-rules list"). Listing actually only returns the current
// contents of that cache.
//
// Note that because this uses the GCLOUD API, GCE staging is unable to use this
// target type. See b/26320525 for more on this.
//
// TODO: The cache layer provided by this, instances, lameduck, and resolver
//               are all pretty similar. RTC will need a similar cache. I should
//               abstract out this whole cache layer. It will be more testable that
//               way, and probably more readable, as well.
type forwardingRules struct {
	project     string
	names       []string
	localRegion string
	cache       map[string]*compute.ForwardingRule
	apiVersion  string
	l           *logger.Logger
}

// List produces a list of all the forwarding rules. The list is similar to
// "gcloud compute forwarding-rules list", but with a cache layer reducing the
// number of actual API calls made.
func (frp *forwardingRules) List() []string {
	return frp.names
}

// Resolve returns the IP address associated with the forwarding
// rule. Eventually we can expand this to return protocol and port as well.
func (frp *forwardingRules) Resolve(name string, ipVer int) (net.IP, error) {
	f := frp.cache[name]
	if f == nil {
		return nil, fmt.Errorf("gce.forwardingRulesProvider.resolve(%s): forwarding rule not in in-memory GCE forwardingRules database", name)
	}
	return net.ParseIP(f.IPAddress), nil
}

// listForwardingRules runs equivalent API calls as "gcloud compute
// forwarding-rules list", and is what is used to populate the cache.
func listForwardingRules(project, apiVersion, region string) ([]*compute.ForwardingRule, error) {
	client, err := google.DefaultClient(oauth2.NoContext, compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	cs.BasePath = "https://www.googleapis.com/compute/" + apiVersion + "/projects/"
	l, err := cs.ForwardingRules.List(project, region).Do()
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

// expand will refill the cache, and update names.
func (frp *forwardingRules) expand() {
	frp.l.Infof("gce.forwardingRules.expand: expanding GCE targets")

	forwardingRules, err := listForwardingRules(frp.project, frp.apiVersion, frp.localRegion)
	if err != nil {
		frp.l.Errorf("gce.forwardingRules.expand: error while getting list of all forwardingRules: %v", err)
		return
	}
	var result []string
	for _, ins := range forwardingRules {
		frp.cache[ins.Name] = ins
		result = append(result, ins.Name)
	}

	frp.l.Debugf("Expanded target list: %q", result)
	frp.names = result
}

// Instance's region is not stored in the metadata, we need to get it from the zone.
func getLocalRegion() (string, error) {
	zone, err := metadata.Zone()
	if err != nil {
		return "", err
	}
	zoneParts := strings.Split(zone, "-")
	return strings.Join(zoneParts[0:len(zoneParts)-1], "-"), nil
}

// newForwardingrules will (if needed) initialize and return the
// globalForwardingRules singleton.
func newForwardingRules(project string, opts *GlobalOptions, l *logger.Logger) (*forwardingRules, error) {
	reEvalInterval := time.Duration(opts.GetReEvalSec()) * time.Second

	var localRegion string
	var err error
	// Initialize forwardingRules provider only once
	onceForwardingRules.Do(func() {
		localRegion, err = getLocalRegion()
		if err != nil {
			err = fmt.Errorf("gce.newForwardingRules: error while getting local region: %v", err)
			return
		}
		l.Infof("gce.newForwardingRules: local region: %s", localRegion)
		globalForwardingRules = &forwardingRules{
			project:     project,
			localRegion: localRegion,
			cache:       make(map[string]*compute.ForwardingRule),
			apiVersion:  opts.GetApiVersion(),
			l:           l,
		}
		go func() {
			globalForwardingRules.expand()
			for _ = range time.Tick(reEvalInterval) {
				globalForwardingRules.expand()
			}
		}()
	})
	if err != nil {
		return nil, err
	}
	return globalForwardingRules, err
}
