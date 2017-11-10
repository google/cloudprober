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
	"sort"
	"sync"
)

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

// AddInt64 generates a panic for the Map type. This is added only to satisfy
// the Value interface.
func (m *Map) AddInt64(i int64) {
	panic("Map type doesn't implement AddInt64()")
}

// AddFloat64 generates a panic for the Map type. This is added only to
// satisfy the Value interface.
func (m *Map) AddFloat64(f float64) {
	panic("Map type doesn't implement AddFloat64()")
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
