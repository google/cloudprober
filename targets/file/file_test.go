package file

import (
	"io/ioutil"
	"reflect"
	"testing"

	rdspb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/targets/endpoint"
	configpb "github.com/google/cloudprober/targets/file/proto"
	"google.golang.org/protobuf/proto"
)

var testResourcesTextpb = `
resource {
  name: "switch-xx-1"
	ip: "10.1.1.1"
	port: 8080
	labels {
	  key: "device_type"
		value: "switch"
	}
	labels {
	  key: "cluster"
		value: "xx"
	}
}
resource {
  name: "switch-xx-2"
	ip: "10.1.1.2"
	port: 8081
	labels {
	  key: "cluster"
		value: "xx"
	}
}
resource {
  name: "switch-yy-1"
	ip: "10.1.2.1"
	port: 8080
}
resource {
  name: "switch-zz-1"
	ip: "::aaa:1"
	port: 8080
}
`

var testResourcesJSON = `{
	"resource": [
		{
			"name": "switch-xx-1",
			"ip": "10.1.1.1",
			"port": 8080,
			"labels": {
				"device_type": "switch",
				"cluster": "xx"
			}
		},
		{
			"name": "switch-xx-2",
			"ip": "10.1.1.2",
			"port": 8081,
			"labels": {
				"cluster": "xx"
			}
		},
		{
			"name": "switch-yy-1",
			"ip": "10.1.2.1",
			"port": 8080
		},
		{
			"name": "switch-zz-1",
			"ip": "::aaa:1",
			"port": 8080
		}
	]
}
`

var testResourcesFile = map[string]string{
	"textpb": testResourcesTextpb,
	"json":   testResourcesJSON,
}

var testExpectedEndpoints = []endpoint.Endpoint{
	{
		Name: "switch-xx-1",
		Port: 8080,
		Labels: map[string]string{
			"device_type": "switch",
			"cluster":     "xx",
		},
	},
	{
		Name: "switch-xx-2",
		Port: 8081,
		Labels: map[string]string{
			"cluster": "xx",
		},
	},
	{
		Name: "switch-yy-1",
		Port: 8080,
	},
	{
		Name: "switch-zz-1",
		Port: 8080,
	},
}

var testExpectedEndpointsWithFilter = []endpoint.Endpoint{
	{
		Name: "switch-xx-1",
		Port: 8080,
		Labels: map[string]string{
			"device_type": "switch",
			"cluster":     "xx",
		},
	},
	{
		Name: "switch-xx-2",
		Port: 8081,
		Labels: map[string]string{
			"cluster": "xx",
		},
	},
}

var testExpectedIP = map[string]string{
	"switch-xx-1": "10.1.1.1",
	"switch-xx-2": "10.1.1.2",
	"switch-yy-1": "10.1.2.1",
	"switch-zz-1": "::aaa:1",
}

func TestFileTargets(t *testing.T) {
	for _, filetype := range []string{"textpb", "json"} {
		t.Run("Test "+filetype, func(t *testing.T) {
			testFileTargetsForType(t, filetype)
		})
	}
}

func createTestFile(t *testing.T, fileType string) string {
	t.Helper()

	if fileType == "" {
		fileType = "textpb"
	}

	tempFile, err := ioutil.TempFile("", "file_targets_*."+fileType)
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(tempFile.Name(), []byte(testResourcesFile[fileType]), 0644); err != nil {
		t.Fatal(err)
	}

	return tempFile.Name()
}

func testFileTargetsForType(t *testing.T, fileType string) {
	t.Helper()
	testFile := createTestFile(t, fileType)

	ft, err := New(&configpb.TargetsConf{FilePath: proto.String(testFile)}, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error while parsing textpb: %v", err)
	}

	t.Log(ft.names)

	got := ft.ListEndpoints()

	if !reflect.DeepEqual(got, testExpectedEndpoints) {
		t.Errorf("ft.ListEndpoints: got: %v, expected: %v", got, testExpectedEndpoints)
	}

	for name, ip := range testExpectedIP {
		resolvedIP, err := ft.Resolve(name, 0)
		if err != nil {
			t.Errorf("unexpected error while resolving %s: %v", name, err)
		}
		got := resolvedIP.String()
		if got != ip {
			t.Errorf("ft.Resolve(%s): got=%s, expected=%s", name, got, ip)
		}
	}
}

func TestListEndpointsWithFilter(t *testing.T) {
	t.Helper()

	testFile := createTestFile(t, "")

	ft, err := New(&configpb.TargetsConf{
		FilePath: proto.String(testFile),
		Filter: []*rdspb.Filter{
			{
				Key:   proto.String("labels.cluster"),
				Value: proto.String("xx"),
			},
		},
	}, nil, nil)

	if err != nil {
		t.Fatalf("Unexpected error while parsing textpb: %v", err)
	}

	t.Log(ft.names)

	got := ft.ListEndpoints()

	if !reflect.DeepEqual(got, testExpectedEndpointsWithFilter) {
		t.Errorf("ft.ListEndpoints: got: %v, expected: %v", got, testExpectedEndpointsWithFilter)
	}
}
