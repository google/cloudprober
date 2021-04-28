// Copyright 2017 The Cloudprober Authors.
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
	"strconv"
	"sync/atomic"
)

// Int implements NumValue with int64 storage. Note that Int is not concurrency
// safe, if you want a concurrency safe integer NumValue, use AtomicInt.
type Int struct {
	i int64
	// If Str is defined, this is method used to convert Int into a string.
	Str func(int64) string
}

// NewInt returns a new Int
func NewInt(i int64) *Int {
	return &Int{i: i}
}

// Clone returns a copy the receiver Int
func (i *Int) Clone() Value {
	return &Int{
		i:   i.i,
		Str: i.Str,
	}
}

// Int64 returns the stored int64
func (i *Int) Int64() int64 {
	return i.i
}

// Float64 returns the stored int64 as a float64
func (i *Int) Float64() float64 {
	return float64(i.i)
}

// Inc increments the receiver Int by one.
// It's part of the NumValue interface.
func (i *Int) Inc() {
	i.i++
}

// IncBy increments the receiver Int by "delta" NumValue.
// It's part of the NumValue interface.
func (i *Int) IncBy(delta NumValue) {
	i.i += delta.Int64()
}

// Add adds a Value to the receiver Int. If Value is not Int, an error is returned.
// It's part of the Value interface.
func (i *Int) Add(val Value) error {
	delta, ok := val.(*Int)
	if !ok {
		return errors.New("incompatible value to add")
	}
	i.i += delta.i
	return nil
}

// AddInt64 adds an int64 to the receiver Int.
func (i *Int) AddInt64(ii int64) {
	i.i += ii
}

// AddFloat64 adds a float64 to the receiver Int.
func (i *Int) AddFloat64(f float64) {
	i.i += int64(f)
}

// String returns the string representation of Int.
// It's part of the Value interface.
func (i *Int) String() string {
	if i.Str != nil {
		return i.Str(i.Int64())
	}
	return strconv.FormatInt(i.Int64(), 10)
}

// AtomicInt implements NumValue with int64 storage and atomic operations. If concurrency-safety
// is not a requirement, e.g. for use in already mutex protected map, you could use Int.
type AtomicInt struct {
	i int64
	// If Str is defined, this is method used to convert AtomicInt into a string.
	Str func(int64) string
}

// NewAtomicInt returns a new AtomicInt
func NewAtomicInt(i int64) *AtomicInt {
	return &AtomicInt{i: i}
}

// Clone returns a copy the receiver AtomicInt
func (i *AtomicInt) Clone() Value {
	return &AtomicInt{
		i:   i.Int64(),
		Str: i.Str,
	}
}

// Int64 returns the stored int64
func (i *AtomicInt) Int64() int64 {
	return atomic.LoadInt64(&i.i)
}

// Float64 returns the stored int64 as a float64
func (i *AtomicInt) Float64() float64 {
	return float64(atomic.LoadInt64(&i.i))
}

// Inc increments the receiver AtomicInt by one.
// It's part of the NumValue interface.
func (i *AtomicInt) Inc() {
	atomic.AddInt64(&i.i, 1)
}

// IncBy increments the receiver AtomicInt by "delta" NumValue.
// It's part of the NumValue interface.
func (i *AtomicInt) IncBy(delta NumValue) {
	atomic.AddInt64(&i.i, delta.Int64())
}

// Add adds a Value to the receiver AtomicInt. If Value is not AtomicInt, an error is returned.
// It's part of the Value interface.
func (i *AtomicInt) Add(val Value) error {
	delta, ok := val.(NumValue)
	if !ok {
		return errors.New("incompatible value to add")
	}
	atomic.AddInt64(&i.i, delta.Int64())
	return nil
}

// AddInt64 adds an int64 to the receiver Int.
func (i *AtomicInt) AddInt64(ii int64) {
	atomic.AddInt64(&i.i, ii)
}

// AddFloat64 adds a float64 to the receiver Int.
func (i *AtomicInt) AddFloat64(f float64) {
	atomic.AddInt64(&i.i, int64(f))
}

// String returns the string representation of AtomicInt.
// It's part of the Value interface.
func (i *AtomicInt) String() string {
	if i.Str != nil {
		return i.Str(i.Int64())
	}
	return strconv.FormatInt(i.Int64(), 10)
}
