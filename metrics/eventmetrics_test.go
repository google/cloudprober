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
	"fmt"
	"testing"
	"time"
)

func newEventMetrics(sent, rcvd, rtt int64, respCodes map[string]int64) *EventMetrics {
	respCodesVal := NewMap("code", NewInt(0))
	for k, v := range respCodes {
		respCodesVal.IncKeyBy(k, NewInt(v))
	}
	em := NewEventMetrics(time.Now()).
		AddMetric("sent", NewInt(sent)).
		AddMetric("rcvd", NewInt(rcvd)).
		AddMetric("rtt", NewInt(rtt)).
		AddMetric("resp-code", respCodesVal)
	return em
}

func verifyOrder(em *EventMetrics, names ...string) error {
	keys := em.MetricsKeys()
	for i := range names {
		if keys[i] != names[i] {
			return fmt.Errorf("Metrics not in order. At Index: %d, Expected: %s, Got: %s", i, names[i], keys[i])
		}
	}
	return nil
}

func verifyEventMetrics(t *testing.T, m *EventMetrics, sent, rcvd, rtt int64, respCodes map[string]int64) {
	// Verify that metrics are ordered correctly.
	if err := verifyOrder(m, "sent", "rcvd", "rtt", "resp-code"); err != nil {
		t.Error(err)
	}

	expectedMetrics := map[string]int64{
		"sent": sent,
		"rcvd": rcvd,
		"rtt":  rtt,
	}
	for k, eVal := range expectedMetrics {
		if m.Metric(k).(NumValue).Int64() != eVal {
			t.Errorf("Unexpected metric value. Expected: %d, Got: %d", eVal, m.Metric(k).(*Int).Int64())
		}
	}
	for k, eVal := range respCodes {
		if m.Metric("resp-code").(*Map).GetKey(k).Int64() != eVal {
			t.Errorf("Unexpected metric value. Expected: %d, Got: %d", eVal, m.Metric("resp-code").(*Map).GetKey(k).Int64())
		}
	}
}

func TestEventMetricsUpdate(t *testing.T) {
	rttVal := NewInt(0)
	rttVal.Str = func(i int64) string {
		return fmt.Sprintf("%.3f", float64(i)/1000)
	}
	m := newEventMetrics(0, 0, 0, make(map[string]int64))
	m.AddLabel("ptype", "http")

	m2 := newEventMetrics(32, 22, 220100, map[string]int64{
		"200": 22,
	})
	m.Update(m2)
	// We'll verify later that mClone is un-impacted by further updates.
	mClone := m.Clone()

	// Verify that "m" has been updated correctly.
	verifyEventMetrics(t, m, 32, 22, 220100, map[string]int64{
		"200": 22,
	})

	m3 := newEventMetrics(30, 30, 300100, map[string]int64{
		"200": 22,
		"204": 8,
	})
	m.Update(m3)

	// Verify that "m" has been updated correctly.
	verifyEventMetrics(t, m, 62, 52, 520200, map[string]int64{
		"200": 44,
		"204": 8,
	})

	// Verify that even though "m" has changed, mClone is as m was after first update
	verifyEventMetrics(t, mClone, 32, 22, 220100, map[string]int64{
		"200": 22,
	})

	// Log metrics in string format
	// t.Log(m.String())

	expectedString := fmt.Sprintf("%d labels=ptype=http sent=62 rcvd=52 rtt=520200 resp-code=map:code,200:44,204:8", m.Timestamp.Unix())
	s := m.String()
	if s != expectedString {
		t.Errorf("em.String()=%s, want=%s", s, expectedString)
	}
}

func BenchmarkEventMetricsStringer(b *testing.B) {
	em := newEventMetrics(32, 22, 220100, map[string]int64{
		"200": 22,
	})
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		em.String()
	}
}
