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

/*
Package runconfig stores cloudprober config that is specific to a single
invocation. e.g., servers injected by external cloudprober users.
*/
package runconfig

import (
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

// runConfig stores cloudprober config that is specific to a single invocation.
// e.g., servers injected by external cloudprober users.
type runConfig struct {
	sync.RWMutex
	grpcSrv *grpc.Server
	version string
}

var rc runConfig

// SetDefaultGRPCServer sets the default gRPC server.
func SetDefaultGRPCServer(s *grpc.Server) error {
	rc.Lock()
	defer rc.Unlock()
	if rc.grpcSrv != nil {
		return fmt.Errorf("gRPC server already set to %v", rc.grpcSrv)
	}
	rc.grpcSrv = s
	return nil
}

// DefaultGRPCServer returns the configured gRPC server and nil if gRPC server
// was not set.
func DefaultGRPCServer() *grpc.Server {
	rc.Lock()
	defer rc.Unlock()
	return rc.grpcSrv
}

// SetVersion sets the cloudprober version.
func SetVersion(version string) {
	rc.Lock()
	defer rc.Unlock()
	rc.version = version
}

// Version returns the runconfig version set through the SetVersion() function
// call. It's useful only if called after SetVersion(), otherwise it will
// return an empty string.
func Version() string {
	rc.RLock()
	defer rc.RUnlock()
	return rc.version
}
