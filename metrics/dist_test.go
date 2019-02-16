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
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	distpb "github.com/google/cloudprober/metrics/proto"
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
	testDistProto := &distpb.Dist{}
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

func TestNewExponentialDistribution(t *testing.T) {
	rows := []struct {
		name              string
		base, scaleFactor float64
		numBuckets        int
		expectedLB        []float64
		wantError         bool
	}{
		{
			name:        "Base:2,SF:1,NB:6",
			base:        2,
			scaleFactor: 1,
			numBuckets:  6,
			expectedLB:  []float64{math.Inf(-1), 0, 1, 2, 4, 8, 16, 32},
		},
		{
			name:        "Base:2,SF:.01,NB:7",
			base:        2,
			scaleFactor: 0.01,
			numBuckets:  7,
			expectedLB:  []float64{math.Inf(-1), 0, .01, .02, .04, .08, .16, .32, .64},
		},
		{
			name:        "Base too low - Base:1,SF:.01,NB:7",
			base:        1,
			scaleFactor: 0.01,
			numBuckets:  7,
			wantError:   true,
		},
	}

	for _, testRow := range rows {
		d, err := NewExponentialDistribution(testRow.base, testRow.scaleFactor, testRow.numBuckets)
		if (err != nil) != testRow.wantError {
			t.Errorf("Row %s: error %v, want error is %v", testRow.name, err, testRow.wantError)
		}
		if len(testRow.expectedLB) != 0 {
			if !reflect.DeepEqual(d.lowerBounds, testRow.expectedLB) {
				t.Errorf("Unexpected lower bounds for exponential distribution. d.lowerBounds=%v, want=%v.", d.lowerBounds, testRow.expectedLB)
			}
		}
	}
}

func TestDistAddSample(t *testing.T) {
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

func TestDistAdd(t *testing.T) {
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

func TestDistData(t *testing.T) {
	lb := []float64{1, 5, 15, 30, 45}
	d := NewDistribution(lb)

	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}

	dd := d.Data()
	want := &DistributionData{
		LowerBounds:  []float64{math.Inf(-1), 1, 5, 15, 30, 45},
		BucketCounts: []int64{1, 1, 0, 1, 0, 0},
		Count:        3,
		Sum:          21.5,
	}
	if !reflect.DeepEqual(dd, want) {
		t.Errorf("Didn't get expected data. d.Data()=%v, want: %v", dd, want)
	}
}

func TestDistString(t *testing.T) {
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

func TestVerify(t *testing.T) {
	d := &Distribution{}
	if d.Verify() == nil {
		t.Fatalf("Distribution verification didn't fail for an invalid distribution.")
	}

	// Now a valid distribution
	lb := []float64{1, 5, 15, 30, 45}
	d = NewDistribution(lb)
	if d.Verify() != nil {
		t.Fatalf("Distribution verification failed for a valid distribution: %s", d.String())
	}

	// Make it invalid by removing one element from the lower bounds.
	d.lowerBounds = d.lowerBounds[1:]
	if d.Verify() == nil {
		t.Fatalf("Distribution verification didn't fail for an invalid distribution: %s.", d.String())
	}

	// Invalid distribution due to count mismatch.
	d = NewDistribution(lb)
	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}
	d.count--
	if d.Verify() == nil {
		t.Fatalf("Distribution verification didn't fail for an invalid distribution (count mismatch): %s.", d.String())
	}
}

func TestParseDistFromString(t *testing.T) {
	lb := []float64{1, 5, 15, 30, 45}
	d := NewDistribution(lb)

	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}

	s := d.String()
	d1, err := ParseDistFromString(s)
	if err != nil {
		t.Fatalf("Error while parsing distribution from: %s. Err: %v", d.String(), err)
	}
	if d1.String() != d.String() {
		t.Errorf("Didn't get the expected distribution. Got: %s, want: %s", d1.String(), d.String())
	}

	// Verify that parsing an invalid string results in error.
	if _, err = ParseDistFromString(strings.Replace(s, "count:3", "count:a", 1)); err == nil {
		t.Error("No error while parsing invalid distribution string.")
	}
}

func BenchmarkDictStringer(b *testing.B) {
	lb := []float64{1, 5, 15, 30, 45}
	d := NewDistribution(lb)

	for _, s := range []float64{0.5, 4, 17} {
		d.AddSample(s)
	}

	// run the d.String() function b.N times
	for n := 0; n < b.N; n++ {
		_ = d.String()
	}
}
