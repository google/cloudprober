// Copyright 2018 Google Inc.
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

// Package formatutils provides web related utils for the cloudprober
// sub-packages.
package formatutils

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

// ConfToString tries to convert the given conf object into a string.
func ConfToString(conf interface{}) string {
	if msg, ok := conf.(proto.Message); ok {
		return proto.MarshalTextString(msg)
	}
	if stringer, ok := conf.(fmt.Stringer); ok {
		return stringer.String()
	}
	return ""
}
