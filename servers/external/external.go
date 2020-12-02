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

// Package external adds support for an external server.
package external

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/servers/external/proto"
	"github.com/google/shlex"
)

// Server implements a external command runner.
type Server struct {
	c *configpb.ServerConf
	l *logger.Logger

	cmdName string
	cmdArgs []string
	cmd     *exec.Cmd
}

// TODO
//	1. Export health status (pid-file OR by monitoring the process started.)
//	2. Add support for command-line substitution.
//  2. Add auto-restart support.

// New creates a new external server.
func New(initCtx context.Context, c *configpb.ServerConf, l *logger.Logger) (*Server, error) {
	cmdParts, err := shlex.Split(c.GetCommand())
	if err != nil {
		return nil, fmt.Errorf("error parsing command line (%s): %v", c.GetCommand(), err)
	}

	return &Server{
		c:       c,
		l:       l,
		cmdName: cmdParts[0],
		cmdArgs: cmdParts[1:len(cmdParts)],
	}, nil
}

// Start runs the external command.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	s.cmd = exec.CommandContext(ctx, s.cmdName, s.cmdArgs...)
	go func() {
		err := s.cmd.Run()
		s.l.Infof("Command %s started. Err status: %v", s.c.GetCommand(), err)
	}()

	return nil
}
