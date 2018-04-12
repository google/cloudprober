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

// Package file implements "file" surfacer. This surfacer type is in
// experimental phase right now.
package file

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"

	configpb "github.com/google/cloudprober/surfacers/file/proto"
)

// FileSurfacer structures for writing onto a GCE instance's serial port. Keeps
// track of an output file which the incoming data is serialized onto (one entry
// per line).
type FileSurfacer struct {
	// Configuration
	c *configpb.SurfacerConf

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Output file for serializing to
	outf *os.File

	// Cloud logger
	l *logger.Logger

	// Each output message has a unique id. This field keeps the record of
	// that.
	id int64
}

// New initializes a FileSurfacer for serializing data into a file (usually set
// as a GCE instance's serial port). This Surfacer does not utilize the Google
// cloud logger because it is unlikely to fail reportably after the call to
// New.
func New(config *configpb.SurfacerConf, l *logger.Logger) (*FileSurfacer, error) {
	s := &FileSurfacer{
		c: config,
		l: l,
	}

	// Get a unique id from the nano timestamp. This id is
	// used to uniquely identify the data strings on the
	// serial port. Only requirement for this id is that it
	// should only go up for a particular instance. We don't
	// call time.Now().UnixNano() for each string that we
	// print as it's an expensive call and we don't really
	// make use of its value.
	id := time.Now().UnixNano()

	return s, s.init(id)
}

func (s *FileSurfacer) init(id int64) error {
	s.writeChan = make(chan *metrics.EventMetrics, 1000)
	s.id = id

	// File handle for the output file
	if s.c.GetFilePath() == "" {
		s.outf = os.Stdout
	} else {
		if outf, err := os.Create(s.c.GetFilePath()); err != nil {
			return fmt.Errorf("failed to create file for writing: %v", err)
		} else {
			s.outf = outf
		}
	}

	// Start a goroutine to run forever, polling on the writeChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go func() {
		for {
			// Write the EventMetrics to file as string.
			em := <-s.writeChan
			if _, err := fmt.Fprintf(s.outf, "%s %d %s\n", s.c.GetPrefix(), s.id, em.String()); err != nil {
				s.l.Warningf("Unable to write data to %s. Err: %v", s.c.GetFilePath(), err)
			}
			s.id++
		}
	}()

	return nil
}

// Write takes the data to be written to file (usually set as a GCE instance's
// serial port). This channel is watched by a goroutine that actually writes
// data to a file.
func (s *FileSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Warningf("FileSurfacer's write channel is full, dropping new data.")
	}
}
