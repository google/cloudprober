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

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// Surfacer structures for writing onto a GCE instance's serial port. Keeps
// track of an output file which the incoming data is serialized onto (one entry
// per line).
type Surfacer struct {
	// Configuration
	c *SurfacerConf

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Output file for serializing to
	outf *os.File

	// Cloud logger
	l *logger.Logger
}

// New initializes a Surfacer for serializing data into a file (usually set as
// a GCE instance's serial port). This Surfacer does not utilize the Google
// cloud logger because it is unlikely to fail reportably after the call to
// New.
// TODO: consider plumbing in a context for the logger
func New(config *SurfacerConf, l *logger.Logger) (*Surfacer, error) {
	// Create an empty surfacer to be returned, assign it an empty write
	// channel to allow for asynch writes.
	s := Surfacer{
		writeChan: make(chan *metrics.EventMetrics, 1000),
		c:         config,
		l:         l,
	}

	// Create an output file to the serial port
	if s.c.GetFilePath() == "" {
		return nil, fmt.Errorf("blank file path provided, please provide a valid file path")
	}
	var err error
	if s.outf, err = os.Create(s.c.GetFilePath()); err != nil {
		s.outf = os.Stdout
		return nil, fmt.Errorf("failed to create file for writing: %v", err)
	}

	// Start the writeTask function to run forever, polling on the writeChan.
	// Allows for the surfacer to write asynchronously to the serial port.
	go func() {
		for {
			s.writeTask()
		}
	}()

	return &s, nil
}

// Write takes the data to be written to file (usually set as a GCE instance's
// serial port). This channel is watched by writeTask and will serialize the
// data portion to file.
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Warningf("Surfacer's write channel is full, dropping new data.")
	}
}

// writeTask polls on the writeChan, and when a message is written to the
// writeChan (using the Write() function) then it converts the message into
// a JSON object and writes it to file as a JSON encoded string.
func (s *Surfacer) writeTask() {
	em := <-s.writeChan

	// Write the EventMetrics to file as string.
	_, err := fmt.Fprintln(s.outf, em.String())
	if err != nil {
		s.l.Warningf("Unable to write data to %s. Err: %v", s.c.GetFilePath(), err)
	}
}
