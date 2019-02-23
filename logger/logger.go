// Copyright 2017-2019 Google Inc.
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

// Package logger provides a logger that logs to Google Cloud Logging. It's a thin wrapper around
// golang/cloud/logging package.
package logger

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/logging"
	"flag"
	"github.com/golang/glog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	monpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

var (
	debugLog = flag.Bool("debug_log", false, "Whether to output debug logs or not")
)

const (
	// Prefix for the cloudprober stackdriver log names.
	cloudproberPrefix = "cloudprober."
)

const (
	// Regular Expression for all characters that are illegal for log names
	//	Ref: https://cloud.google.com/logging/docs/api/ref_v2beta1/rest/v2beta1/LogEntry
	disapprovedRegExp = "[^A-Za-z0-9_/.-]"

	// MaxLogEntrySize Max value of each log entry size
	MaxLogEntrySize = 4096
)

// Logger implements a logger that logs messages to Google Cloud Logging. It provides a suite
// of methods where each method correponds to a specific logging.Level, e.g.
// Error(paylod interface{}). Each method takes a payload that has to either be a
// JSON-encodable object, a string or a []byte slice (all other types of payload will result
// in error).
//
// It falls back to logging through the traditional logger if:
//
//   * Not running on GCE,
//   * Logging client is uninitialized (e.g. for testing),
//   * Logging to cloud fails for some reason.
//
// Logger{} is a valid object that will log through the traditional logger.
//
type Logger struct {
	logc   *logging.Client
	logger *logging.Logger
	// TODO(manugarg): Logger should eventually embed the probe id and each probe
	// should get a different Logger object (embedding that probe's probe id) but
	// sharing the same logging client. We could then make probe id one of the
	// metadata on all logging messages.
}

// NewCloudproberLog is a convenient wrapper around New that sets context to
// context.Background and attaches cloudprober prefix to log names.
func NewCloudproberLog(component string) (*Logger, error) {
	return New(context.Background(), cloudproberPrefix+component)
}

// New returns a new Logger object with cloud logging client initialized if running on GCE.
func New(ctx context.Context, logName string) (*Logger, error) {
	l := &Logger{}

	if !metadata.OnGCE() {
		return l, nil
	}
	projectID, err := metadata.ProjectID()
	if err != nil {
		return nil, err
	}

	instanceID, err := metadata.InstanceID()
	if err != nil {
		return nil, err
	}

	zone, err := metadata.Zone()
	if err != nil {
		return nil, err
	}

	// Check for illegal characters in the log name
	if match, err := regexp.Match(disapprovedRegExp, []byte(logName)); err != nil || match {
		if err != nil {
			return nil, fmt.Errorf("unable to parse logName: %v", err)
		}
		return nil, fmt.Errorf("logName of %s contains an invalid character, valid characters are [A-Za-z0-9_/.-]", logName)
	}
	// Any forward slashes need to be URL encoded, so we query escape to replace them
	logName = url.QueryEscape(logName)

	l.logc, err = logging.NewClient(ctx, projectID, option.WithTokenSource(google.ComputeTokenSource("")))
	if err != nil {
		return nil, err
	}
	l.logger = l.logc.Logger(logName,
		logging.CommonResource(&monpb.MonitoredResource{
			Type: "gce_instance",
			Labels: map[string]string{
				"project_id":  projectID,
				"instance_id": instanceID,
				"zone":        zone,
			},
		}),
		// Encourage batching of write requests.
		// Flush logs to remote logging after 1000 entries (default is 10).
		logging.EntryCountThreshold(1000),
		// Maximum amount of time that an item should remain buffered in memory
		// before being flushed to the logging service. Default is 1 second.
		// We want flushing to be mostly driven by the buffer size (configured
		// above), rather than time.
		logging.DelayThreshold(10*time.Second),
	)
	return l, nil
}

func payloadToString(payload ...string) string {
	if len(payload) == 1 {
		return payload[0]
	}

	var b strings.Builder
	for _, s := range payload {
		b.WriteString(s)
	}
	return b.String()
}

// log sends payload ([]string) to cloud logging. If cloud logging client is
// not initialized (e.g. if not running on GCE) or cloud logging fails for some
// reason, it writes logs through the traditional logger.
func (l *Logger) log(severity logging.Severity, payload ...string) {
	payloadStr := payloadToString(payload...)

	if len(payloadStr) > MaxLogEntrySize {
		truncateMsg := "... (truncated)"
		truncateMsgLen := len(truncateMsg)
		payloadStr = payloadStr[:MaxLogEntrySize-truncateMsgLen] + truncateMsg
	}

	if l == nil || l.logc == nil {
		genericLog(severity, payloadStr)
		return
	}

	l.logger.Log(logging.Entry{
		Severity: severity,
		Payload:  payloadStr,
	})
}

// close closes the cloud logging client if it exists. This flushes the buffer
// and should be called before exiting the program to ensure all logs are persisted.
func (l *Logger) close() error {
	if l != nil && l.logc != nil {
		return l.logc.Close()
	}

	return nil
}

// Debug logs messages with logging level set to "Debug".
func (l *Logger) Debug(payload ...string) {
	if *debugLog {
		l.log(logging.Debug, payload...)
	}
}

// Info logs messages with logging level set to "Info".
func (l *Logger) Info(payload ...string) {
	l.log(logging.Info, payload...)
}

// Warning logs messages with logging level set to "Warning".
func (l *Logger) Warning(payload ...string) {
	l.log(logging.Warning, payload...)
}

// Error logs messages with logging level set to "Error".
func (l *Logger) Error(payload ...string) {
	l.log(logging.Error, payload...)
}

// Critical logs messages with logging level set to "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Critical(payload ...string) {
	l.log(logging.Critical, payload...)
	if err := l.close(); err != nil {
		panic(fmt.Sprintf("could not close client: %v", err))
	}
	os.Exit(1)
}

// Debugf logs formatted text messages with logging level "Debug".
func (l *Logger) Debugf(format string, args ...interface{}) {
	if *debugLog {
		l.log(logging.Debug, fmt.Sprintf(format, args...))
	}
}

// Infof logs formatted text messages with logging level "Info".
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(logging.Info, fmt.Sprintf(format, args...))
}

// Warningf logs formatted text messages with logging level "Warning".
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.log(logging.Warning, fmt.Sprintf(format, args...))
}

// Errorf logs formatted text messages with logging level "Error".
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(logging.Error, fmt.Sprintf(format, args...))
}

// Criticalf logs formatted text messages with logging level "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.log(logging.Critical, fmt.Sprintf(format, args...))
	if err := l.close(); err != nil {
		panic(fmt.Sprintf("could not close client: %v", err))
	}
	os.Exit(1)
}

func genericLog(severity logging.Severity, s string) {
	// Set the caller frame depth to 3 so that can get to the actual caller of
	// the logger. genericLog -> log -> Info* -> actualCaller
	depth := 3

	switch severity {
	case logging.Debug, logging.Info:
		glog.InfoDepth(depth, s)
	case logging.Warning:
		glog.WarningDepth(depth, s)
	case logging.Error:
		glog.ErrorDepth(depth, s)
	case logging.Critical:
		glog.FatalDepth(depth, s)
	}
}
