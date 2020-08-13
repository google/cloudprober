// Copyright 2020 Google Inc.
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

// Package pubsub implements the "pubsub" surfacer. This surfacer type is in
// experimental phase right now.
package pubsub

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/common/compress"
	"github.com/google/cloudprober/sysvars"

	configpb "github.com/google/cloudprober/surfacers/pubsub/proto"
)

const (
	inChanCapacity = 1000
	publishTimeout = 10 * time.Second
)

var newPubsubClient = func(ctx context.Context, project string) (*pubsub.Client, error) {
	return pubsub.NewClient(ctx, project)
}

// Surfacer implements a pubsub surfacer.
type Surfacer struct {
	// Configuration
	c *configpb.SurfacerConf

	// Channel for incoming data.
	inChan            chan *metrics.EventMetrics
	publishResultChan chan *pubsub.PublishResult

	topic      *pubsub.Topic
	topicName  string
	gcpProject string

	l                 *logger.Logger
	starttime         string
	compressionBuffer *compress.CompressionBuffer
	processInputWg    sync.WaitGroup
}

func (s *Surfacer) publishMessage(globalCtx context.Context, data []byte) {
	boolToString := map[bool]string{
		true:  "true",
		false: "false",
	}
	msg := &pubsub.Message{
		Attributes: map[string]string{
			"compressed": boolToString[s.c.GetCompressionEnabled()],
			"starttime":  s.starttime,
		},
		Data: data,
	}

	publishCtx, cancel := context.WithTimeout(globalCtx, publishTimeout)
	defer cancel()
	s.publishResultChan <- s.topic.Publish(publishCtx, msg)
}

func (s *Surfacer) processInput(ctx context.Context) {
	defer s.processInputWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		// Publish the EventMetrics to the topic as a pubsub message.
		case em, ok := <-s.inChan:
			if !ok {
				return
			}
			if s.c.GetCompressionEnabled() {
				s.compressionBuffer.WriteLineToBuffer(em.String())
			} else {
				s.publishMessage(ctx, []byte(em.String()))
			}
		}
	}
}

func (s *Surfacer) init(ctx context.Context) error {
	s.inChan = make(chan *metrics.EventMetrics, inChanCapacity)

	// We use start timestamp in millisecond as the incarnation id.
	s.starttime = strconv.FormatInt(time.Now().UnixNano()/(1000*1000), 10)

	if s.topicName == "" {
		s.topicName = "cloudprober-" + sysvars.Vars()["hostname"]
	}

	if s.gcpProject == "" && metadata.OnGCE() {
		project, err := metadata.ProjectID()
		if err != nil {
			return fmt.Errorf("pubsub_surfacer: unable to retrieve project id: %v", err)
		}
		s.gcpProject = project
	}

	client, err := newPubsubClient(ctx, s.gcpProject)
	if err != nil {
		return fmt.Errorf("pubsub_surfacer: error creating pubsub client: %v", err)
	}

	s.topic = client.Topic(s.topicName)
	exists, err := s.topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("pubsub_surfacer: error determining if topic (%s) exists: %v", s.topicName, err)
	}

	if !exists {
		topic, err := client.CreateTopic(ctx, s.topicName)
		if err != nil {
			return fmt.Errorf("pubsub_surfacer: error creating topic (%s) for publishing: %v", s.topicName, err)
		}
		s.topic = topic
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.topic.Stop()
				return
			case res, ok := <-s.publishResultChan:
				if !ok {
					return
				}
				_, err := res.Get(ctx)
				if err != nil {
					s.l.Warningf("Error publishing message: %v", err)
				}
			}
		}
	}()

	if s.c.GetCompressionEnabled() {
		s.compressionBuffer = compress.NewCompressionBuffer(ctx, func(data []byte) {
			s.publishMessage(ctx, data)
		}, s.l)
	}

	// Start a goroutine to run forever, polling on the inChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	s.processInputWg.Add(1)
	go s.processInput(ctx)

	return nil
}

// close closes the input channel, waits for input processing to finish,
// and closes the compression buffer if open.
func (s *Surfacer) close() {
	close(s.inChan)
	s.processInputWg.Wait()

	if s.compressionBuffer != nil {
		s.compressionBuffer.Close()
	}
	close(s.publishResultChan)
	s.topic.Stop()
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually publishes it to a pubsub topic.
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.inChan <- em:
	default:
		s.l.Errorf("Surfacer's write channel (capacity: %d) is full, dropping new data.", inChanCapacity)
	}
}

// New initializes a Surfacer for publishing data to a pubsub topic.
func New(ctx context.Context, config *configpb.SurfacerConf, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c:                 config,
		l:                 l,
		topicName:         config.GetTopicName(),
		gcpProject:        config.GetProject(),
		publishResultChan: make(chan *pubsub.PublishResult, 1000),
	}

	return s, s.init(ctx)
}
