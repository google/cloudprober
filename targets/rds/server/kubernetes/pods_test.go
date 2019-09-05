package kubernetes

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
)

func TestListResources(t *testing.T) {
	pl := &podsLister{}
	pl.names = []string{"podA", "podB", "podC"}
	pl.cache = map[string]*resource{
		"podA": &resource{name: "podA", namespace: "nsAB", labels: map[string]string{"app": "appA"}, ip: "10.1.1.1"},
		"podB": &resource{name: "podB", namespace: "nsAB", labels: map[string]string{"app": "appB"}, ip: "10.1.1.2"},
		"podC": &resource{name: "podC", namespace: "nsC", labels: map[string]string{"app": "appC", "func": "web"}, ip: "10.1.1.3"},
	}

	tests := []struct {
		desc         string
		nameFilter   string
		filters      map[string]string
		labelsFilter map[string]string
		wantPods     []string
		wantErr      bool
	}{
		{
			desc:    "bad filter key, expect error",
			filters: map[string]string{"names": "pod(B|C)"},
			wantErr: true,
		},
		{
			desc:     "only name filter for podB and podC",
			filters:  map[string]string{"name": "pod(B|C)"},
			wantPods: []string{"podB", "podC"},
		},
		{
			desc:     "name and namespace filter for podB",
			filters:  map[string]string{"name": "pod(B|C)", "namespace": "nsAB"},
			wantPods: []string{"podB"},
		},
		{
			desc:     "only namespace filter for podA and podB",
			filters:  map[string]string{"namespace": "nsAB"},
			wantPods: []string{"podA", "podB"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var filtersPB []*pb.Filter
			for k, v := range test.filters {
				filtersPB = append(filtersPB, &pb.Filter{Key: proto.String(k), Value: proto.String(v)})
			}

			results, err := pl.listResources(filtersPB)
			if err != nil {
				if !test.wantErr {
					t.Errorf("got unexpected error: %v", err)
				}
				return
			}

			var gotNames []string
			for _, res := range results {
				gotNames = append(gotNames, res.GetName())
			}

			if !reflect.DeepEqual(gotNames, test.wantPods) {
				t.Errorf("pods.listResources: got=%v, expected=%v", gotNames, test.wantPods)
			}
		})
	}
}
