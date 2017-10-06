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

// Package serverutils provides utilities to work with the cloudprober's external probe.
package serverutils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

func readPayload(r *bufio.Reader) ([]byte, error) {
	// header format is: "\nContent-Length: %d\n\n"
	const prefix = "Content-Length: "
	var line string
	var length int
	var err error

	// Read lines until header line is found
	for {
		line, err = r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(line, prefix) {
			break
		}
	}

	// Parse content length from the header
	length, err = strconv.Atoi(line[len(prefix) : len(line)-1])
	if err != nil {
		return nil, err
	}
	// Consume the blank line following the header line
	if _, err = r.ReadSlice('\n'); err != nil {
		return nil, err
	}

	// Slurp in the payload
	buf := make([]byte, length)
	if _, err = io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadProbeReply reads ProbeReply from the supplied bufio.Reader and returns it to
// the caller.
func ReadProbeReply(r *bufio.Reader) (*ProbeReply, error) {
	buf, err := readPayload(r)
	if err != nil {
		return nil, err
	}
	rep := new(ProbeReply)
	return rep, proto.Unmarshal(buf, rep)
}

// ReadProbeRequest reads and parses ProbeRequest protocol buffers from the given
// bufio.Reader.
func ReadProbeRequest(r *bufio.Reader) (*ProbeRequest, error) {
	buf, err := readPayload(r)
	if err != nil {
		return nil, err
	}
	req := new(ProbeRequest)
	return req, proto.Unmarshal(buf, req)
}

// WriteMessage marshals the a proto message and writes it to the writer "w"
// with appropriate Content-Length header.
func WriteMessage(pb proto.Message, w io.Writer) error {
	buf, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("Failed marshalling proto message: %v", err)
	}
	if _, err := fmt.Fprintf(w, "\nContent-Length: %d\n\n%s", len(buf), buf); err != nil {
		return fmt.Errorf("Failed writing response: %v", err)
	}
	return nil
}

// Serve blocks indefinitely, servicing probe requests. Note that this function is
// provided mainly to help external probe server implementations. Cloudprober doesn't
// make use of it. Example usage:
//	import (
//		serverpb "github.com/google/cloudprober/probes/external/serverutils/server_proto"
//		"github.com/google/cloudprober/probes/external/serverutils/serverutils"
//	)
//	func runProbe(opts []*cppb.ProbeRequest_Option) {
//  	...
//	}
//	serverutils.Serve(func(req *ProbeRequest, reply *ProbeReply) {
// 		payload, errMsg, _ := runProbe(req.GetOptions())
//		reply.Payload = proto.String(payload)
//		if errMsg != "" {
//			reply.ErrorMessage = proto.String(errMsg)
//		}
//	})
func Serve(probeFunc func(*ProbeRequest, *ProbeReply)) {
	stdin := bufio.NewReader(os.Stdin)

	repliesChan := make(chan *ProbeReply)

	// Write replies to stdout. These are not required to be in-order.
	go func() {
		for rep := range repliesChan {
			if err := WriteMessage(rep, os.Stdout); err != nil {
				log.Fatal(err)
			}

		}
	}()

	// Read requests from stdin, and dispatch probes to service them.
	for {
		request, err := ReadProbeRequest(stdin)
		if err != nil {
			log.Fatalf("Failed reading request: %v", err)
		}
		go func() {
			reply := &ProbeReply{
				RequestId: request.RequestId,
			}
			done := make(chan bool, 1)
			timeout := time.After(time.Duration(*request.TimeLimit) * time.Millisecond)
			go func() {
				probeFunc(request, reply)
				done <- true
			}()
			select {
			case <-done:
				repliesChan <- reply
			case <-timeout:
				// drop the request on the floor.
				fmt.Fprintf(os.Stderr, "Timeout for request %v\n", *reply.RequestId)
			}
		}()
	}
}
