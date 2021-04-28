// Copyright 2018 The Cloudprober Authors.
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

package gcp

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/gcp/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	"google.golang.org/api/option"
)

/*
PubSubFilters defines filters supported by the pubsub_messages resource type.
 Example:
 filter {
	 key: "subscription"
	 value: "sub1.*"
 }
 filter {
	 key: "updated_within"
	 value: "5m"
 }
*/
var PubSubFilters = struct {
	RegexFilterKeys    []string
	FreshnessFilterKey string
}{
	[]string{"subscription"},
	"updated_within",
}

// pubsubMsgsLister is a PubSub Messages lister. It implements a cache,
// that's populated at a regular interval by making the GCP API calls.
// Listing actually only returns the current contents of that cache.
type pubsubMsgsLister struct {
	project    string
	c          *configpb.PubSubMessages
	apiVersion string
	l          *logger.Logger

	mu     sync.RWMutex // Mutex for names and cache
	cache  map[string]map[string]time.Time
	subs   map[string]*pubsub.Subscription
	maxAge time.Duration
	client *pubsub.Client
}

// listResources returns the list of resource records, where each record
// consists of a PubSub message name.
func (lister *pubsubMsgsLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	allFilters, err := filter.ParseFilters(req.GetFilter(), PubSubFilters.RegexFilterKeys, PubSubFilters.FreshnessFilterKey)
	if err != nil {
		return nil, err
	}

	subFilter, freshnessFilter := allFilters.RegexFilters["subscription"], allFilters.FreshnessFilter

	lister.mu.RLock()
	defer lister.mu.RUnlock()

	type msg struct {
		name      string
		timestamp int64
	}
	var result []msg
	for subName, msgs := range lister.cache {
		if subFilter != nil && !subFilter.Match(subName, lister.l) {
			continue
		}

		for name, publishTime := range msgs {
			if freshnessFilter != nil && !freshnessFilter.Match(publishTime, lister.l) {
				continue
			}
			result = append(result, msg{name, publishTime.Unix()})
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	resources := make([]*pb.Resource, len(result))
	for i, msg := range result {
		resources[i] = &pb.Resource{
			Name:        proto.String(msg.name),
			LastUpdated: proto.Int64(msg.timestamp),
		}
	}
	return resources, nil
}

func (lister *pubsubMsgsLister) initSubscriber(ctx context.Context, sub *configpb.PubSubMessages_Subscription) (*pubsub.Subscription, error) {
	s := lister.client.Subscription(sub.GetName())
	seekBackDuration := time.Duration(int(sub.GetSeekBackDurationSec())) * time.Second

	ok, err := s.Exists(ctx)
	if err != nil {
		return nil, err
	}

	// If subscriber exists already, seek back by seek_back_duration_sec.
	if ok {
		err := s.SeekToTime(ctx, time.Now().Add(-seekBackDuration))
		if err != nil {
			return nil, fmt.Errorf("pubsub: error seeking back time for the subscription: %s", sub.GetName())
		}
		return s, nil
	}

	s, err = lister.client.CreateSubscription(ctx, sub.GetName(), pubsub.SubscriptionConfig{
		Topic:               lister.client.Topic(sub.GetTopicName()),
		RetainAckedMessages: true,
		RetentionDuration:   seekBackDuration,
		ExpirationPolicy:    24 * time.Hour,
	})

	if err != nil {
		return nil, fmt.Errorf("pubsub: error creating subscription (%s): %v", sub.GetName(), err)
	}

	return s, nil
}

func (lister *pubsubMsgsLister) cleanup() {
	for range time.Tick(lister.maxAge) {
		lister.mu.Lock()

		for _, msgs := range lister.cache {
			for name, publishTime := range msgs {
				if time.Now().Sub(publishTime) > lister.maxAge {
					lister.l.Infof("pubsub.cleanup: deleting the expired message: %s, publish time: %s", name, publishTime)
					delete(msgs, name)
				}
			}
		}

		lister.mu.Unlock()
	}
}

func newPubSubMsgsLister(project string, c *configpb.PubSubMessages, l *logger.Logger) (*pubsubMsgsLister, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if c.GetApiEndpoint() != "" {
		opts = append(opts, option.WithEndpoint(c.GetApiEndpoint()))
	}

	client, err := pubsub.NewClient(ctx, project, opts...)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %v", err)
	}

	lister := &pubsubMsgsLister{
		project: project,
		c:       c,
		cache:   make(map[string]map[string]time.Time),
		client:  client,
		subs:    make(map[string]*pubsub.Subscription),
		l:       l,
		maxAge:  time.Hour,
	}

	// Start a cleanup goroutine to remove expired entries from the in-memory
	// storage.
	go lister.cleanup()

	for _, sub := range lister.c.GetSubscription() {
		s, err := lister.initSubscriber(ctx, sub)
		if err != nil {
			return nil, err
		}

		name := sub.GetName()
		lister.subs[name] = s
		lister.cache[name] = make(map[string]time.Time)

		lister.l.Infof("pubsub: Receiving pub/sub messages for project (%s) and subscription (%s)", lister.project, name)
		go s.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			lister.mu.Lock()
			defer lister.mu.Unlock()

			lister.l.Infof("pubsub: Adding message with name: %s, message id: %s, publish time: %s", msg.Attributes["name"], msg.ID, msg.PublishTime)
			lister.cache[name][msg.Attributes["name"]] = msg.PublishTime
			msg.Ack()
		})
	}

	return lister, nil
}
