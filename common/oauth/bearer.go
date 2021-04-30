// Copyright 2019 The Cloudprober Authors.
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

package oauth

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/common/file"
	configpb "github.com/google/cloudprober/common/oauth/proto"
	"github.com/google/cloudprober/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type bearerTokenSource struct {
	c                   *configpb.BearerToken
	getTokenFromBackend func(*configpb.BearerToken) (string, error)
	cache               string
	mu                  sync.RWMutex
	l                   *logger.Logger
}

var getTokenFromFile = func(c *configpb.BearerToken) (string, error) {
	b, err := file.ReadFile(c.GetFile())
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var getTokenFromCmd = func(c *configpb.BearerToken) (string, error) {
	var cmd *exec.Cmd

	cmdParts := strings.Split(c.GetCmd(), " ")
	cmd = exec.Command(cmdParts[0], cmdParts[1:len(cmdParts)]...)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v", cmd)
	}
	return string(out), nil
}

var getTokenFromGCEMetadata = func(c *configpb.BearerToken) (string, error) {
	ts := google.ComputeTokenSource(c.GetGceServiceAccount())
	tok, err := ts.Token()
	return tok.AccessToken, err
}

func newBearerTokenSource(c *configpb.BearerToken, l *logger.Logger) (oauth2.TokenSource, error) {
	ts := &bearerTokenSource{
		c: c,
		l: l,
	}

	switch ts.c.Source.(type) {
	case *configpb.BearerToken_File:
		ts.getTokenFromBackend = getTokenFromFile

	case *configpb.BearerToken_Cmd:
		ts.getTokenFromBackend = getTokenFromCmd

	case *configpb.BearerToken_GceServiceAccount:
		ts.getTokenFromBackend = getTokenFromGCEMetadata

	default:
		ts.getTokenFromBackend = getTokenFromGCEMetadata
	}

	tok, err := ts.getTokenFromBackend(c)
	if err != nil {
		return nil, err
	}
	ts.cache = tok

	if ts.c.GetRefreshIntervalSec() == 0 {
		return ts, nil
	}

	go func() {
		interval := time.Duration(ts.c.GetRefreshIntervalSec()) * time.Second

		for range time.Tick(interval) {
			tok, err := ts.getTokenFromBackend(ts.c)

			if err != nil {
				ts.l.Warningf("oauth.bearerTokenSource: %s", err)
				continue
			}

			ts.mu.Lock()
			ts.cache = tok
			ts.mu.Unlock()
		}
	}()

	return ts, nil
}

func (ts *bearerTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.c.GetRefreshIntervalSec() == 0 {
		tok, err := ts.getTokenFromBackend(ts.c)

		if err != nil {
			if ts.cache == "" {
				return nil, err
			}

			ts.l.Errorf("oauth.bearerTokenSource: failed to get token: %v, using cache", err)
			return &oauth2.Token{AccessToken: ts.cache}, nil
		}

		return &oauth2.Token{AccessToken: tok}, nil
	}

	return &oauth2.Token{AccessToken: ts.cache}, nil
}
