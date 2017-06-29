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

// Package metrics implements data types for probes generated data.
package metrics

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

// Value represents any metric value
type Value interface {
	clone() Value
	Add(delta Value) error
	String() string
}

// NumValue represents any numerical metric value, e.g. Int, Float.
// It's a superset of Value interface.
type NumValue interface {
	Value
	Inc()
	Int64() int64
	IncBy(delta NumValue)
}

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

func (i *Int) clone() Value {
	return &Int{
		i:   i.i,
		Str: i.Str,
	}
}

// Int64 returns the stored int64
func (i *Int) Int64() int64 {
	return i.i
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

func (i *AtomicInt) clone() Value {
	return &AtomicInt{
		i:   i.Int64(),
		Str: i.Str,
	}
}

// Int64 returns the stored int64
func (i *AtomicInt) Int64() int64 {
	return atomic.LoadInt64(&i.i)
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

// String returns the string representation of AtomicInt.
// It's part of the Value interface.
func (i *AtomicInt) String() string {
	if i.Str != nil {
		return i.Str(i.Int64())
	}
	return strconv.FormatInt(i.Int64(), 10)
}

// String implements a value type with string storage.
// It satisfies the Value interface.
type String struct {
	s string
}

// NewString returns a new String with the given string value.
func NewString(s string) String {
	return String{s: s}
}

// Add isn't supported for String type, this is only to satisfy the Value interface.
func (s String) Add(val Value) error {
	return errors.New("string value type doesn't support Add() operation")
}

// String simply returns the stored string.
func (s String) String() string {
	return "\"" + s.s + "\""
}

func (s String) clone() Value {
	return String{s: s.s}
}

// Map implements a key-value store where keys are of type string and values are
// of type NumValue.
// It satisfies the Value interface.
type Map struct {
	MapName string // Map key name
	mu      sync.RWMutex
	m       map[string]NumValue
	keys    []string

	// We use this to initialize new keys
	defaultKeyValue NumValue
}

// NewMap returns a new Map
func NewMap(mapName string, defaultValue NumValue) *Map {
	return &Map{
		MapName:         mapName,
		defaultKeyValue: defaultValue,
		m:               make(map[string]NumValue),
	}
}

// GetKey returns the given key's value.
// TODO: We should probably add a way to get the list of all the keys in the
// map.
func (m *Map) GetKey(key string) NumValue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[key]
}

// clone creates a clone of the Map. clone makes sure that underlying data
// storage is properly cloned.
func (m *Map) clone() Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newMap := &Map{
		MapName:         m.MapName,
		defaultKeyValue: m.defaultKeyValue.clone().(NumValue),
		m:               make(map[string]NumValue),
	}
	newMap.keys = make([]string, len(m.keys))
	for i, k := range m.keys {
		newMap.m[k] = m.m[k].clone().(NumValue)
		newMap.keys[i] = m.keys[i]
	}
	return newMap
}

// Keys returns the list of keys
func (m *Map) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.keys...)
}

// newKey adds a new key to the map, with its value set to defaultKeyValue
// This is an unsafe function, callers should take care of protecting the map
// from race conditions.
func (m *Map) newKey(key string) {
	m.keys = append(m.keys, key)
	sort.Strings(m.keys)
	m.m[key] = m.defaultKeyValue.clone().(NumValue)
}

// IncKey increments the given key's value by one.
func (m *Map) IncKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m[key] == nil {
		m.newKey(key)
	}
	m.m[key].Inc()
}

// IncKeyBy increments the given key's value by NumValue.
func (m *Map) IncKeyBy(key string, delta NumValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m[key] == nil {
		m.newKey(key)
	}
	m.m[key].IncBy(delta)
}

// Add adds a value (type Value) to the receiver Map. A non-Map value returns an error.
// This is part of the Value interface.
func (m *Map) Add(val Value) error {
	delta, ok := val.(*Map)
	if !ok {
		return errors.New("incompatible value to add")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delta.mu.RLock()
	defer delta.mu.RUnlock()
	var sortRequired bool
	for k, v := range delta.m {
		if m.m[k] == nil {
			sortRequired = true
			m.keys = append(m.keys, k)
			m.m[k] = v
			continue
		}
		m.m[k].Add(v)
	}
	if sortRequired {
		sort.Strings(m.keys)
	}
	return nil
}

// String returns the string representation of the receiver Map.
// This is part of the Value interface.
func (m *Map) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := fmt.Sprintf("map:%s", m.MapName)
	for _, k := range m.keys {
		s = fmt.Sprintf("%s,%s:%s", s, k, m.m[k].String())
	}
	return s
}
