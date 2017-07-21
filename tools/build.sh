#!/bin/bash -eu
#
# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script builds cloudprober from source. It expects protobuf's Go code to
# be already available (can be done using tools/gen_pb_go.sh).

PROTOC_VERSION="3.3.0"
PROJECT="github.com/google/cloudprober"

GOPATH=$(go env GOPATH)

if [ -z "$GOPATH" ]; then
  echo "Go environment is not setup correctly. Please look at"
  echo "https://golang.org/doc/code.html to set up Go environment."
  exit 1
fi

# Change directory to the project workspace in GOPATH
project_dir="${GOPATH}/src/${PROJECT}"

if [ ! -d "${project_dir}" ];then
  echo "${PROJECT} not found under Go workspace: ${GOPATH}/src. Please download"
  echo " cloudprober source code from github.com/google/cloudprober and set it "
  echo "such that it's available at ${project_dir}."
  exit 1
fi

cd ${project_dir}
# Get all dependencies
echo "Getting all the dependencies.."
echo "======================================================================"
go get -t ./...

# Build everything
echo "Build everything..."
echo "======================================================================"
go build ./...

# Run tests
echo "Running tests..."
echo "======================================================================"
go test ./...

# Install cloudprober
echo "Build static cloudprober binary.."
echo "======================================================================"
CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' ./cmd/cloudprober.go
