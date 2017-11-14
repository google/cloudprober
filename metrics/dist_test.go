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

package metrics

import (
	"math"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
)

func verifyBucketCount(t *testing.T, d *Distribution, indices []int, counts []int64) {
	for _, i := range indices {
		if d.bucketCounts[i] != counts[i] {
			t.Errorf("For bucket with index %d and lower bound: %f, expected count: %d, got: %d", i, d.lowerBounds[i], counts[i], d.bucketCounts[i])
			t.Logf("Dist: %v", d.bucketCounts)
		}
	}
}

func protoToDist(t *testing.T, testDistProtoText string) *Distribution {
	testDistProto := &Dist{}
	if err := proto.UnmarshalText(testDistProtoText, testDistProto); err != nil {
		t.Errorf("Failed parsing distribution proto text: %s. Err: %v", testDistProtoText, err)
		return nil
	}
	d, err := NewDistributionFromProto(testDistProto)
	if err != nil {
		t.Errorf("Error while creating distrubtion from the protobuf: %s. Err: %v", testDistProtoText, err)
		return nil
	}
	return d
}

func TestNewDistributionFromProto(t *testing.T) {
	testDistProtoText := `
	  explicit_buckets: "1,2,4,8,16,32"
	`
	expectedLowerBounds := []float64{math.Inf(-1), 1, 2, 4, 8, 16, 32}
	d := protoToDist(t, testDistProtoText)
	if !reflect.DeepEqual(d.lowerBounds, expectedLowerBounds) {
		t.Errorf("Unexpected lower bounds from proto. d.lowerBounds=%v, want=%v.", d.lowerBounds, expectedLowerBounds)
	}
}

func TestAddSample(t *testing.T) {
	lb := []float64{1, 5, 10, 15, 20, 30, 40, 50}
	d := NewDistribution(lb)

	if len(d.lowerBounds) != len(lb)+1 || len(d.bucketCounts) != len(lb)+1 {
		t.Errorf("Distribution object not properly formed. Dist: %v", d)
		t.FailNow()
	}

	for _, s := range []float64{0.5, 4, 17, 21, 3, 300} {
		d.AddSample(s)
	}

	verifyBucketCount(t, d, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, []int64{1, 2, 0, 0, 1, 1, 0, 0, 1})

	t.Log(d.String())
}

func TestAdd(t *testing.T) {
	lb := []float64{1, 5, 15, 30, 45}
	d := NewDistribution(lb)

	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}

	delta := NewDistribution(lb)
	for _, s := range []float64{3.5, 21, 300} {
		delta.AddSample(s)
	}

	if err := d.Add(delta); err != nil {
		t.Error(err)
	}
	verifyBucketCount(t, d, []int{0, 1, 2, 3, 4, 5}, []int64{1, 2, 0, 2, 0, 1})
}

func TestString(t *testing.T) {
	lb := []float64{1, 5, 15, 30, 45}
	d := NewDistribution(lb)

	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}

	s := d.String()
	want := "dist:sum:21.5|count:3|lb:-Inf,1,5,15,30,45|bc:1,1,0,1,0,0"
	if s != want {
		t.Errorf("String is not in expected format. d.String()=%s, want: %s", s, want)
	}
}
