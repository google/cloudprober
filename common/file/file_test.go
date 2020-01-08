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

package file

import (
	"io/ioutil"
	"testing"
)

func createTempFile(t *testing.T, b []byte) string {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
		return ""
	}

	defer tmpfile.Close()
	if _, err := tmpfile.Write(b); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

func testReadFile(path string) ([]byte, error) {
	return []byte("content-for-" + path), nil
}

func TestReadFile(t *testing.T) {
	prefixToReadfunc["test://"] = testReadFile

	// Virtual file
	testPath := "test://test-file"

	// Disk file
	tempContent := "temp-content"
	tempPath := createTempFile(t, []byte(tempContent))

	testData := map[string]string{
		testPath: "content-for-test-file",
		tempPath: tempContent,
	}

	for path, expectedContent := range testData {
		t.Run("ReadFile("+path+")", func(t *testing.T) {
			b, err := ReadFile(path)
			if err != nil {
				t.Fatalf("Error while reading the file: %s", path)
			}

			if string(b) != expectedContent {
				t.Errorf("ReadFile(%s) = %s, expected=%s", path, string(b), expectedContent)
			}
		})
	}
}
