// Copyright 2017-2020 The Cloudprober Authors.
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

// Package file implements the "file" surfacer.
package file

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/common/compress"
	"github.com/google/cloudprober/surfacers/common/options"

	configpb "github.com/google/cloudprober/surfacers/file/proto"
)

// Surfacer structures for writing onto a GCE instance's serial port. Keeps
// track of an output file which the incoming data is serialized onto (one entry
// per line).
type Surfacer struct {
	// Configuration
	c    *configpb.SurfacerConf
	opts *options.Options

	// Channel for incoming data.
	inChan         chan *metrics.EventMetrics
	processInputWg sync.WaitGroup

	// Output file for serializing to
	outf *os.File

	// Cloud logger
	l *logger.Logger

	// Each output message has a unique id. This field keeps the record of
	// that.
	id int64

	compressionBuffer *compress.CompressionBuffer
}

func (s *Surfacer) processInput(ctx context.Context) {
	defer s.processInputWg.Done()

	if !s.c.GetCompressionEnabled() {
	}
	for {
		select {
		// Write the EventMetrics to file as string.
		case em, ok := <-s.inChan:
			if !ok {
				return
			}
			var emStr strings.Builder
			emStr.WriteString(s.c.GetPrefix())
			emStr.WriteByte(' ')
			emStr.WriteString(strconv.FormatInt(s.id, 10))
			emStr.WriteByte(' ')
			emStr.WriteString(em.String())
			s.id++

			// If compression is not enabled, write line to file and continue.
			if !s.c.GetCompressionEnabled() {
				if _, err := s.outf.WriteString(emStr.String() + "\n"); err != nil {
					s.l.Errorf("Unable to write data to %s. Err: %v", s.c.GetFilePath(), err)
				}
			} else {
				s.compressionBuffer.WriteLineToBuffer(emStr.String())
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *Surfacer) init(ctx context.Context, id int64) error {
	s.inChan = make(chan *metrics.EventMetrics, s.opts.MetricsBufferSize)
	s.id = id

	// File handle for the output file
	if s.c.GetFilePath() == "" {
		s.outf = os.Stdout
	} else {
		outf, err := os.Create(s.c.GetFilePath())
		if err != nil {
			return fmt.Errorf("failed to create file for writing: %v", err)
		}
		s.outf = outf
	}

	if s.c.GetCompressionEnabled() {
		s.compressionBuffer = compress.NewCompressionBuffer(ctx, func(data []byte) {
			if _, err := s.outf.Write(append(data, '\n')); err != nil {
				s.l.Errorf("Unable to write data to %s. Err: %v", s.outf.Name(), err)
			}
		}, s.opts.MetricsBufferSize/10, s.l)
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

	s.outf.Close()
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually writes data to a file ((usually set as a GCE
// instance's serial port).
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.inChan <- em:
	default:
		s.l.Errorf("Surfacer's write channel is full, dropping new data.")
	}
}

// New initializes a Surfacer for serializing data into a file (usually set
// as a GCE instance's serial port). This Surfacer does not utilize the Google
// cloud logger because it is unlikely to fail reportably after the call to
// New.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c:    config,
		opts: opts,
		l:    l,
	}

	// Get a unique id from the nano timestamp. This id is
	// used to uniquely identify the data strings on the
	// serial port. Only requirement for this id is that it
	// should only go up for a particular instance. We don't
	// call time.Now().UnixNano() for each string that we
	// print as it's an expensive call and we don't really
	// make use of its value.
	id := time.Now().UnixNano()

	return s, s.init(ctx, id)
}
