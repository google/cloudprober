// Copyright 2017-2018 Google Inc.
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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"

	configpb "github.com/google/cloudprober/surfacers/file/proto"
)

var (
	compressionBufferFlushInterval = time.Second
	compressionBufferMaxLines      = 100
)

// FileSurfacer structures for writing onto a GCE instance's serial port. Keeps
// track of an output file which the incoming data is serialized onto (one entry
// per line).
type FileSurfacer struct {
	// Configuration
	c *configpb.SurfacerConf

	// Channel for incoming data.
	inChan chan *metrics.EventMetrics

	// Output file for serializing to
	outf *os.File

	// Cloud logger
	l *logger.Logger

	// Each output message has a unique id. This field keeps the record of
	// that.
	id int64

	compressionBuffer *compressionBuffer
}

// compressionBuffer stores the data that is ready to be compressed.
type compressionBuffer struct {
	sync.Mutex
	buf     bytes.Buffer
	lines   int
	l       *logger.Logger
	outChan chan string
}

func newCompressionBuffer(ctx context.Context, outf *os.File, l *logger.Logger) *compressionBuffer {
	c := &compressionBuffer{
		outChan: make(chan string, 1000),
		l:       l,
	}

	// Start a loop to read econded strings from the outChan channel and write them
	// to the output file.
	go func() {
		for {
			select {
			case str := <-c.outChan:
				if _, err := fmt.Fprintln(outf, str); err != nil {
					c.l.Errorf("Unable to write data to %s. Err: %v", outf.Name(), err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start the flush loop: call flush every sec, until ctx.Done().
	go func() {
		ticker := time.NewTicker(compressionBufferFlushInterval)
		for {
			select {
			case <-ticker.C:
				c.flush()
			case <-ctx.Done():
				return
			}
		}
	}()

	return c
}

func (c *compressionBuffer) writeLine(line string) {
	triggerFlush := false

	c.Lock()
	c.buf.WriteString(line + "\n")
	c.lines++
	if c.lines >= compressionBufferMaxLines {
		triggerFlush = true
	}
	c.Unlock()

	// triggerFlush is decided within the locked section.
	if triggerFlush {
		c.flush()
	}
}

func compressBytes(inBytes []byte) (string, error) {
	var outBuf bytes.Buffer
	b64w := base64.NewEncoder(base64.StdEncoding, &outBuf)
	gw := gzip.NewWriter(b64w)
	if _, err := gw.Write(inBytes); err != nil {
		return "", err
	}
	gw.Close()
	b64w.Close()

	return outBuf.String(), nil
}

// flush compresses the data in buffer and writes it to outChan.
func (c *compressionBuffer) flush() {
	// Retrieve bytes from the buffer (c.buf) and reset it.
	c.Lock()
	inBytes := c.buf.Bytes()
	c.buf.Reset()
	c.lines = 0
	c.Unlock()

	// Nothing to do.
	if len(inBytes) == 0 {
		return
	}

	compressed, err := compressBytes(inBytes)
	if err != nil {
		c.l.Errorf("Error while compressing bytes: %v, data: %s", err, string(inBytes))
		return
	}
	c.outChan <- compressed
}

// New initializes a FileSurfacer for serializing data into a file (usually set
// as a GCE instance's serial port). This Surfacer does not utilize the Google
// cloud logger because it is unlikely to fail reportably after the call to
// New.
func New(config *configpb.SurfacerConf, l *logger.Logger) (*FileSurfacer, error) {
	// Create a context.
	ctx := context.TODO()

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

	return s, s.init(ctx, id)
}

func (s *FileSurfacer) processInput(ctx context.Context) {
	for {
		select {
		// Write the EventMetrics to file as string.
		case em := <-s.inChan:
			emStr := fmt.Sprintf("%s %d %s", s.c.GetPrefix(), s.id, em.String())
			s.id++

			// If compression is not enabled, write line to file and continue.
			if !s.c.GetCompressionEnabled() {
				if _, err := fmt.Fprintln(s.outf, emStr); err != nil {
					s.l.Errorf("Unable to write data to %s. Err: %v", s.c.GetFilePath(), err)
				}
				continue
			}
			s.compressionBuffer.writeLine(emStr)

		case <-ctx.Done():
			return
		}
	}
}

func (s *FileSurfacer) init(ctx context.Context, id int64) error {
	s.inChan = make(chan *metrics.EventMetrics, 1000)
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
		s.compressionBuffer = newCompressionBuffer(ctx, s.outf, s.l)
	}

	// Start a goroutine to run forever, polling on the inChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go s.processInput(ctx)

	return nil
}

// Write queues the incoming data into a channel. This channel is watched by a
// goroutine that actually writes data to a file ((usually set as a GCE
// instance's serial port).
func (s *FileSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.inChan <- em:
	default:
		s.l.Errorf("FileSurfacer's write channel is full, dropping new data.")
	}
}
