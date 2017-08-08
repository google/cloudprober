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

package rtcservice

import (
	"fmt"
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

const (
	Write = iota
	Delete
)

type Action int

// Small regression tests for the RTC Stub object. Checks that read, writes,
// and deletes all are working.
func TestStub(t *testing.T) {
	s := NewStub()
	var actions = []struct {
		act         Action
		key         string
		val         string
		expectError bool
		want        []string
	}{
		{Write, "k1", "v1", false,
			[]string{"k1 : v1"}},
		{Write, "k2", "v2", false,
			[]string{"k1 : v1", "k2 : v2"}},
		{Write, "k1", "v2", false,
			[]string{"k1 : v2", "k2 : v2"}},
		{Delete, "k2", "", false,
			[]string{"k1 : v2"}},
		{Delete, "k3", "", true,
			[]string{"k1 : v2"}},
	}

	for id, a := range actions {
		// Perform action
		var err error
		switch a.act {
		case Write:
			err = s.Write(a.key, []byte(a.val))
		case Delete:
			err = s.Delete(a.key)
		default:
			t.Error("In test row ", id, ": a.act = %v, want Write (%v) or Delete (%v).", a.act, Write, Delete)
			continue
		}

		// Check for error
		if (err != nil) != a.expectError {
			t.Error("In test row ", id, ": Action err = %v, want %v.", err, a.expectError)
			continue
		}

		// Check all rtc vars
		gotVars, err := s.List()
		if err != nil {
			t.Error("In test row ", id, ": s.List() err = %v, want nil. ", err)
		}
		got := make([]string, len(gotVars))
		for i, g := range gotVars {
			k := g.Name
			v, err := s.Val(g)
			if err != nil {
				t.Errorf("In test row %v : s.Val(%#v) erred. %v", id, g, err)
				continue
			}
			got[i] = fmt.Sprintf("%v : %v", k, string(v))
		}
		sort.Strings(got)
		sort.Strings(a.want)
		if diff := pretty.Compare(a.want, got); diff != "" {
			t.Errorf("In test row %v : rtc.List() got diff:\n%s", id, diff)
		}
	}
}
