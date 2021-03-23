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

// Package compress implements compression related utilities.
package compress

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
)

var (
	compressionBufferFlushInterval = time.Second
)

// CompressionBuffer stores the data that is ready to be compressed.
type CompressionBuffer struct {
	mu        sync.Mutex
	buf       *bytes.Buffer
	lines     int
	batchSize int
	l         *logger.Logger

	// Function to callback with compressed data.
	callback func([]byte)

	// Cancel function for the internal context, to stop the compression and
	// flushing loop.
	cancelCtx context.CancelFunc
}

// NewCompressionBuffer returns a new compression buffer.
func NewCompressionBuffer(inctx context.Context, callback func([]byte), batchSize int, l *logger.Logger) *CompressionBuffer {
	ctx, cancel := context.WithCancel(inctx)
	c := &CompressionBuffer{
		buf:       new(bytes.Buffer),
		callback:  callback,
		batchSize: batchSize,
		l:         l,
		cancelCtx: cancel,
	}

	// Start the flush loop: call flush every sec, until ctx.Done().
	go func() {
		ticker := time.NewTicker(compressionBufferFlushInterval)
		defer ticker.Stop()

		for range ticker.C {
			select {
			case <-ctx.Done():
				return
			default:
			}

			c.compressAndCallback()
		}
	}()

	return c
}

// WriteLineToBuffer writes the given line to the buffer.
func (c *CompressionBuffer) WriteLineToBuffer(line string) {
	triggerFlush := false

	c.mu.Lock()
	c.buf.WriteString(line)
	c.buf.WriteString("\n")
	c.lines++
	if c.lines >= c.batchSize {
		triggerFlush = true
	}
	c.mu.Unlock()

	// triggerFlush is decided within the locked section.
	if triggerFlush {
		c.compressAndCallback()
	}
}

// Compress compresses the given bytes and encodes them into a string.
func Compress(inBytes []byte) ([]byte, error) {
	var outBuf bytes.Buffer
	b64w := base64.NewEncoder(base64.StdEncoding, &outBuf)
	gw := gzip.NewWriter(b64w)
	if _, err := gw.Write(inBytes); err != nil {
		return nil, err
	}
	gw.Close()
	b64w.Close()

	return outBuf.Bytes(), nil
}

// compressAndCallback compresses the data in buffer and writes it to outChan.
func (c *CompressionBuffer) compressAndCallback() {
	// Retrieve bytes from the buffer (c.buf) and get a new buffer for c.
	c.mu.Lock()
	inBytes := c.buf.Bytes()

	// Start c's new buf with the same capacity as old buf.
	c.buf = bytes.NewBuffer(make([]byte, 0, c.buf.Cap()))
	c.lines = 0
	c.mu.Unlock()

	// Nothing to do.
	if len(inBytes) == 0 {
		return
	}

	compressed, err := Compress(inBytes)
	if err != nil {
		c.l.Errorf("Error while compressing bytes: %v, data: %s", err, string(inBytes))
		return
	}
	c.callback(compressed)
}

// Close compresses the buffer and flushes it to the output channel.
func (c *CompressionBuffer) Close() {
	c.cancelCtx()
	c.compressAndCallback()
}
