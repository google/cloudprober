// Copyright 2021 Google Inc.
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

package targets

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/cloudprober/targets/endpoint"
)

func staticTargets(hosts string) (Targets, error) {
	t, _ := baseTargets(nil, nil, nil)
	sl := &staticLister{}

	for _, host := range strings.Split(hosts, ",") {
		host = strings.TrimSpace(host)

		// Make sure there is no "/" in the host name. That typically happens
		// when users accidentally add URLs in hostnames.
		if strings.IndexByte(host, '/') >= 0 {
			return nil, fmt.Errorf("invalid host (%s), contains '/'", host)
		}

		hostColonParts := strings.Split(host, ":")

		// There is no colon in host name.
		if len(hostColonParts) == 1 {
			sl.list = append(sl.list, endpoint.Endpoint{Name: host})
			continue
		}

		// There is only 1 colon, assume it is for the port. An IPv6 address will
		// more than 1 colon.
		if len(hostColonParts) == 2 {
			portNum, err := strconv.Atoi(hostColonParts[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing port(%s): %v", hostColonParts[1], err)
			}
			sl.list = append(sl.list, endpoint.Endpoint{Name: hostColonParts[0], Port: portNum})
			continue
		}

		// More than 1 colon. It should include an IPv6 address.
		// 1. Parses as an IP address. If that fails,
		// 2. Parse for IPv6 and port.
		if ip := net.ParseIP(host); ip != nil {
			sl.list = append(sl.list, endpoint.Endpoint{Name: host})
			continue
		}
		hostname, port, err := net.SplitHostPort(host)
		if err != nil {
			return nil, fmt.Errorf("error parsing host(%s) as hostport: %v", host, err)
		}
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("error parsing port(%s): %v", port, err)
		}
		sl.list = append(sl.list, endpoint.Endpoint{Name: hostname, Port: portNum})
	}

	t.lister = sl
	t.resolver = globalResolver
	return t, nil
}
