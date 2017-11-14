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

	"google.golang.org/api/googleapi"
	"google.golang.org/api/monitoring/v3"
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
func NewDistributionFromProto(distProto *Dist) (*Distribution, error) {
	switch distProto.Buckets.(type) {
	case *Dist_ExplicitBuckets:
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
	case *Dist_ExponentialBuckets:
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

func (d *Distribution) clone() Value {
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

// Clone returns a copy of the receiver distribution.
func (d *Distribution) Clone() *Distribution {
	return d.clone().(*Distribution)
}
