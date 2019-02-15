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
// TODO(manugarg): Add more tests after a bit of refactoring.

package lameduck

import (
	"reflect"
	"testing"
)

type mockLDLister struct {
	list []string
}

func (mldl *mockLDLister) List() []string {
	return mldl.list
}

func TestDefaultLister(t *testing.T) {
	list1 := []string{"list1"}

	// Initialize default lister with the given lister
	InitDefaultLister(nil, &mockLDLister{list1}, nil)
	lister, err := GetDefaultLister()
	if err != nil {
		t.Fatal(err)
	}
	gotList := lister.List()
	if !reflect.DeepEqual(gotList, list1) {
		t.Errorf("Default lister retured: %v, expected: %v", gotList, list1)
	}

	list2 := []string{"list2"}
	// Initialize default lister with the given lister. This time it should have
	// no impact as default lister is already initialized.
	InitDefaultLister(nil, &mockLDLister{list2}, nil)
	lister, err = GetDefaultLister()
	if err != nil {
		t.Fatal(err)
	}
	gotList = lister.List()
	if !reflect.DeepEqual(gotList, list1) {
		t.Errorf("Default lister retured: %v, expected: %v", gotList, list1)
	}
}
