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
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	distpb "github.com/google/cloudprober/metrics/proto"
	"google.golang.org/api/googleapi"
	monitoring "google.golang.org/api/monitoring/v3"
)

// Distribution metrics type implements a histogram of values distributed over
// a set of pre-defined buckets.
type Distribution struct {
	mu           sync.RWMutex
	lowerBounds  []float64 // bucket lower bounds
	bucketCounts []int64
	count        int64   // count of all values
	sum          float64 // sum of all samples.
}

// NewDistribution returns a new distribution container.
func NewDistribution(lowerBounds []float64) *Distribution {
	return &Distribution{
		lowerBounds:  append([]float64{math.Inf(-1)}, lowerBounds...),
		bucketCounts: make([]int64, len(lowerBounds)+1),
	}
}

// NewDistributionFromProto returns a new distribution based on the provided
// protobuf.
func NewDistributionFromProto(distProto *distpb.Dist) (*Distribution, error) {
	switch distProto.Buckets.(type) {
	case *distpb.Dist_ExplicitBuckets:
		lbStringA := strings.Split(distProto.GetExplicitBuckets(), ",")
		lowerBounds := make([]float64, len(lbStringA))
		for i, tok := range lbStringA {
			lb, err := strconv.ParseFloat(tok, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid lower bound for bucket: %s. Err: %v", tok, err)
			}
			lowerBounds[i] = lb
		}
		return NewDistribution(lowerBounds), nil
	case *distpb.Dist_ExponentialBuckets:
		return nil, errors.New("exponential buckets are not supported yet")
	}
	return nil, fmt.Errorf("unknown buckets type: %v", distProto.Buckets)
}

func (d *Distribution) bucketIndex(sample float64) int {
	return sort.Search(len(d.lowerBounds), func(i int) bool { return sample < d.lowerBounds[i] }) - 1
}

// AddSample adds a sample to the receiver distribution.
func (d *Distribution) AddSample(sample float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bucketCounts[d.bucketIndex(sample)]++
	d.sum += sample
	d.count++
}

// AddInt64 adds an int64 to the receiver distribution.
func (d *Distribution) AddInt64(i int64) {
	d.AddSample(float64(i))
}

// AddFloat64 adds an float64 to the receiver distribution.
func (d *Distribution) AddFloat64(f float64) {
	d.AddSample(f)
}

// Add adds a distribution to the receiver distribution. If both distributions
// don't have the same buckets, an error is returned.
func (d *Distribution) Add(val Value) error {
	delta, ok := val.(*Distribution)
	if !ok {
		return errors.New("incompatible value to add to distribution")
	}

	if !reflect.DeepEqual(d.lowerBounds, delta.lowerBounds) {
		return fmt.Errorf("incompatible delta value, Bucket lower bounds in receiver distribution: %v, and in delta distribution: %v", d.lowerBounds, delta.lowerBounds)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delta.mu.RLock()
	defer delta.mu.RUnlock()

	for i := 0; i < len(d.bucketCounts); i++ {
		d.bucketCounts[i] += delta.bucketCounts[i]
	}
	d.count += delta.count
	d.sum += delta.sum
	return nil
}

// String returns a string representation of the distribution:
// "dist:sum:<sum>|count:<count>|lb:<lower bounds>|bc:<bucket counts>"
// For example for a distribution with lower bounds 0.5, 2.0, 7.5 and
// bucket counts 34, 54, 121, 12, string representation will look like the
// following:
// dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12
func (d *Distribution) String() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var tokens []string

	tokens = append(tokens, fmt.Sprintf("sum:%s", strconv.FormatFloat(d.sum, 'f', -1, 64)))
	tokens = append(tokens, fmt.Sprintf("count:%d", d.count))

	tok := "lb:"
	for _, lb := range d.lowerBounds {
		tok = fmt.Sprintf("%s%s,", tok, strconv.FormatFloat(lb, 'f', -1, 64))
	}
	tok = tok[:len(tok)-1] // Remove last ","
	tokens = append(tokens, tok)

	tok = "bc:"
	for _, c := range d.bucketCounts {
		tok = fmt.Sprintf("%s%d,", tok, c)
	}
	tok = tok[:len(tok)-1] // Remove last ","
	tokens = append(tokens, tok)

	return "dist:" + strings.Join(tokens, "|")
}

// Verify verifies that the distribution is valid.
func (d *Distribution) Verify() error {
	if len(d.lowerBounds) == 0 {
		return errors.New("no distribution buckets found")
	}
	if len(d.lowerBounds) != len(d.bucketCounts) {
		return fmt.Errorf("size mismatch between buckets array (%v) and bucket counts array (%v)", d.lowerBounds, d.bucketCounts)
	}
	var countSum int64
	for _, c := range d.bucketCounts {
		countSum += c
	}
	if d.count != countSum {
		return fmt.Errorf("sum of bucket counts (%d) don't match with the overall count (%d)", countSum, d.count)
	}
	return nil
}

// ParseDistFromString parses a distribution value from a string that's in a
// format that's generated by the String() method:
// Example string: dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12
func ParseDistFromString(str string) (*Distribution, error) {
	tokens := strings.SplitN(str, ":", 2)
	if len(tokens) != 2 || tokens[0] != "dist" {
		return nil, fmt.Errorf("invalid distribution string: %s", str)
	}

	d := &Distribution{}

	var f float64
	var i int64
	var err error

	errF := func(kv []string, err error) (*Distribution, error) {
		return nil, fmt.Errorf("invalid token (%s:%s) in the distribution string: %s. Err: %v", kv[0], kv[1], str, err)
	}

	for _, tok := range strings.Split(tokens[1], "|") {
		kv := strings.Split(tok, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid distribution string: %s", str)
		}
		switch kv[0] {
		case "sum":
			if f, err = strconv.ParseFloat(kv[1], 64); err != nil {
				return errF(kv, err)
			}
			d.sum = f
		case "count":
			if i, err = strconv.ParseInt(kv[1], 10, 64); err != nil {
				return errF(kv, err)
			}
			d.count = i
		case "lb":
			for _, vs := range strings.Split(kv[1], ",") {
				if f, err = strconv.ParseFloat(vs, 64); err != nil {
					return errF(kv, err)
				}
				d.lowerBounds = append(d.lowerBounds, f)
			}
		case "bc":
			for _, vs := range strings.Split(kv[1], ",") {
				if i, err = strconv.ParseInt(vs, 10, 64); err != nil {
					return errF(kv, err)
				}
				d.bucketCounts = append(d.bucketCounts, i)
			}
		default:
			return errF(kv, nil)
		}
	}
	if err := d.Verify(); err != nil {
		return nil, err
	}
	return d, nil
}

// DistributionData stuct, along with Data() function, provides a way to
// readily share the Distribution data with other packages.
type DistributionData struct {
	LowerBounds  []float64 // bucket lower bounds
	BucketCounts []int64
	Count        int64   // count of all values
	Sum          float64 // sum of all samples.
}

// Data returns a DistributionData object, built using Distribution's current
// state.
func (d *Distribution) Data() *DistributionData {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return &DistributionData{
		LowerBounds:  d.lowerBounds,
		BucketCounts: d.bucketCounts,
		Count:        d.count,
		Sum:          d.sum,
	}
}

// StackdriverTypedValue returns a Stackdriver typed value corresponding to the
// receiver distribution. This routine is used by stackdriver surfacer.
func (d *Distribution) StackdriverTypedValue() *monitoring.TypedValue {
	d.mu.RLock()
	defer d.mu.RUnlock()
	distVal := &monitoring.Distribution{
		BucketCounts: googleapi.Int64s(append([]int64{}, d.bucketCounts...)),
		BucketOptions: &monitoring.BucketOptions{
			ExplicitBuckets: &monitoring.Explicit{
				Bounds: append([]float64{}, d.lowerBounds[1:]...),
			},
		},
		Count: d.count,
	}
	return &monitoring.TypedValue{
		DistributionValue: distVal,
	}
}

// Clone returns a copy of the receiver distribution.
func (d *Distribution) Clone() Value {
	d.mu.RLock()
	defer d.mu.RUnlock()
	newD := NewDistribution(d.lowerBounds[1:])
	newD.sum = d.sum
	newD.count = d.count
	for i := range d.bucketCounts {
		newD.bucketCounts[i] = d.bucketCounts[i]
	}
	return newD
}
