// Copyright 2020 Google Inc.
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
Package file implements utilities to read files from various backends.
*/
package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/oauth2/google"
)

type readFunc func(path string) ([]byte, error)

var prefixToReadfunc = map[string]readFunc{
	"gs://": readFileFromGCS,
}

func readFileFromGCS(objectPath string) ([]byte, error) {
	hc, err := google.DefaultClient(context.Background())
	if err != nil {
		return nil, err
	}

	objURL := "https://storage.googleapis.com/" + objectPath
	res, err := hc.Get(objURL)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got error while retrieving GCS object, http status: %s, status code: %d", res.Status, res.StatusCode)
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// ReadFile returns file contents as a slice of bytes. It's similar to ioutil's
// ReadFile, but includes support for files on non-disk locations. For example,
// files with paths starting with gs:// are assumed to be on GCS, and are read
// from GCS.
func ReadFile(fname string) ([]byte, error) {
	for prefix, f := range prefixToReadfunc {
		if strings.HasPrefix(fname, prefix) {
			return f(fname[len(prefix):])
		}
	}
	return ioutil.ReadFile(fname)
}
