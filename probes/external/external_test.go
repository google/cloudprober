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

package external

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/external/serverutils"
)

func TestPayloadToEventMetrics(t *testing.T) {
	p := &Probe{
		name: "testprobe",
	}
	payload := []string{
		"time_to_running 10",
		"time_to_ssh 30",
	}
	target := "target"
	em := p.payloadToMetrics(target, strings.Join(payload, "\n"))
	expectedEM := metrics.NewEventMetrics(em.Timestamp).
		AddMetric("time_to_running", metrics.NewInt(10)).
		AddMetric("time_to_ssh", metrics.NewInt(30)).
		AddLabel("ptype", "external").
		AddLabel("probe", "testprobe").
		AddLabel("dst", "target")
	if em.String() != expectedEM.String() {
		t.Errorf("payload not parsed correctly.\nExpected: %s\n, Got: %s", expectedEM.String(), em.String())
	}
}

func TestSubstituteLabels(t *testing.T) {
	tests := []struct {
		desc   string
		in     string
		labels map[string]string
		want   string
		found  bool
	}{
		{
			desc:  "No replacement",
			in:    "foo bar baz",
			want:  "foo bar baz",
			found: true,
		},
		{
			desc: "Replacement beginning",
			in:   "@foo@ bar baz",
			labels: map[string]string{
				"foo": "h e llo",
			},
			want:  "h e llo bar baz",
			found: true,
		},
		{
			desc: "Replacement middle",
			in:   "beginning @😿@ end",
			labels: map[string]string{
				"😿": "😺",
			},
			want:  "beginning 😺 end",
			found: true,
		},
		{
			desc: "Replacement end",
			in:   "bar baz @foo@",
			labels: map[string]string{
				"foo": "XöX",
				"bar": "nope",
			},
			want:  "bar baz XöX",
			found: true,
		},
		{
			desc: "Replacements",
			in:   "abc@foo@def@foo@ jk",
			labels: map[string]string{
				"def": "nope",
				"foo": "XöX",
			},
			want:  "abcXöXdefXöX jk",
			found: true,
		},
		{
			desc: "Multiple labels",
			in:   "xx @foo@@bar@ yy",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want:  "xx XöX_ yy",
			found: true,
		},
		{
			desc: "Not found",
			in:   "A b C @d@ e",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want: "A b C @d@ e",
		},
		{
			desc: "@@",
			in:   "hello@@foo",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want:  "hello@foo",
			found: true,
		},
		{
			desc: "odd number",
			in:   "hello@foo@bar@xx",
			labels: map[string]string{
				"foo": "yy",
			},
			want:  "helloyybar@xx",
			found: true,
		},
	}

	for _, tc := range tests {
		got, found := substituteLabels(tc.in, tc.labels)
		if tc.found != found {
			t.Errorf("%v: substituteLabels(%q, %q) = _, %v, want %v", tc.desc, tc.in, tc.labels, found, tc.found)
		}
		if tc.want != got {
			t.Errorf("%v: substituteLabels(%q, %q) = %q, _, want %q", tc.desc, tc.in, tc.labels, got, tc.want)
		}
	}
}

type BufferCloser struct {
	bytes.Buffer
}

func (b *BufferCloser) Close() error { return nil }

func TestSendRequest(t *testing.T) {
	buf := &BufferCloser{}
	p := &Probe{
		name:     "testprobe",
		cmdStdin: buf,
		c: &ProbeConf{
			Options: []*ProbeConf_Option{
				&ProbeConf_Option{
					Name:  proto.String("target"),
					Value: proto.String("@target@"),
				},
			},
		},
	}
	requestID := int32(1234)
	target := "dummy"
	labels := map[string]string{
		"probe":   p.name,
		"target":  target,
		"address": "1.2.3.4",
	}
	err := p.sendRequest(requestID, labels)
	if err != nil {
		t.Errorf("Failed to sendRequest: %v", err)
	}
	req := new(serverutils.ProbeRequest)
	var length int
	_, err = fmt.Fscanf(buf, "\nContent-Length: %d\n\n", &length)
	if err != nil {
		t.Errorf("Failed to read header: %v", err)
	}
	err = proto.Unmarshal(buf.Bytes(), req)
	if err != nil {
		t.Fatalf("Failed to Unmarshal probe Request: %v", err)
	}
	if got, want := req.GetRequestId(), requestID; got != requestID {
		t.Errorf("req.GetRequestId() = %q, want %v", got, want)
	}
	opts := req.GetOptions()
	if len(opts) != 1 {
		t.Errorf("req.GetOptions() = %q (%v), want only one item", opts, len(opts))
	}
	if got, want := opts[0].GetName(), "target"; got != want {
		t.Errorf("opts[0].GetName() = %q, want %q", got, want)
	}
	if got, want := opts[0].GetValue(), target; got != target {
		t.Errorf("opts[0].GetValue() = %q, want %q", got, want)
	}
}
