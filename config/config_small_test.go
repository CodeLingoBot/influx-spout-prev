// Copyright 2017 Jump Trading
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build small

package config

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfigBaseName = "something"

var testConfigFileName = filepath.Join("some", "dir", testConfigBaseName+".toml")

func TestCorrectConfigFile(t *testing.T) {
	const validConfigSample = `
name = "thor"

mode = "listener"
port = 10001

nats_address = "nats://localhost:4222"
nats_subject = ["spout"]
nats_subject_monitor = "spout-monitor"

influxdb_address = "localhost"
influxdb_port = 8086
influxdb_dbname = "junk_nats"

batch = 10
batch_max_mb = 5
batch_max_secs = 60
workers = 96

write_timeout_secs = 32
read_buffer_bytes = 43210
nats_pending_max_mb = 100
listener_batch_bytes = 4096
max_time_delta_secs = 789
`
	conf, err := parseConfig(validConfigSample)
	require.NoError(t, err, "Couldn't parse a valid config: %v\n", err)

	assert.Equal(t, "thor", conf.Name, "Name must match")
	assert.Equal(t, "listener", conf.Mode, "Mode must match")
	assert.Equal(t, 10001, conf.Port, "Port must match")
	assert.Equal(t, 10, conf.BatchMessages, "Batching must match")
	assert.Equal(t, 5, conf.BatchMaxMB)
	assert.Equal(t, 60, conf.BatchMaxSecs)
	assert.Equal(t, 96, conf.Workers)
	assert.Equal(t, 32, conf.WriteTimeoutSecs, "WriteTimeoutSecs must match")
	assert.Equal(t, 43210, conf.ReadBufferBytes)
	assert.Equal(t, 100, conf.NATSPendingMaxMB)
	assert.Equal(t, 4096, conf.ListenerBatchBytes)
	assert.Equal(t, 789, conf.MaxTimeDeltaSecs)

	assert.Equal(t, 8086, conf.InfluxDBPort, "InfluxDB Port must match")
	assert.Equal(t, "junk_nats", conf.DBName, "InfluxDB DBname must match")
	assert.Equal(t, "localhost", conf.InfluxDBAddress, "InfluxDB address must match")

	assert.Equal(t, "spout", conf.NATSSubject[0], "Subject must match")
	assert.Equal(t, "spout-monitor", conf.NATSSubjectMonitor, "Monitor subject must match")
	assert.Equal(t, "nats://localhost:4222", conf.NATSAddress, "Address must match")
}

func TestAllDefaults(t *testing.T) {
	conf, err := parseConfig(`mode = "writer"`)
	require.NoError(t, err)

	assert.Equal(t, testConfigBaseName, conf.Name)
	assert.Equal(t, "nats://localhost:4222", conf.NATSAddress)
	assert.Equal(t, []string{"influx-spout"}, conf.NATSSubject)
	assert.Equal(t, "influx-spout-monitor", conf.NATSSubjectMonitor)
	assert.Equal(t, "influx-spout-junk", conf.NATSSubjectJunkyard)
	assert.Equal(t, "localhost", conf.InfluxDBAddress)
	assert.Equal(t, 8086, conf.InfluxDBPort)
	assert.Equal(t, "influx-spout-junk", conf.DBName)
	assert.Equal(t, 10, conf.BatchMessages)
	assert.Equal(t, 10, conf.BatchMaxMB)
	assert.Equal(t, 300, conf.BatchMaxSecs)
	assert.Equal(t, 0, conf.Port)
	assert.Equal(t, "writer", conf.Mode)
	assert.Equal(t, 8, conf.Workers)
	assert.Equal(t, 30, conf.WriteTimeoutSecs)
	assert.Equal(t, 4194304, conf.ReadBufferBytes)
	assert.Equal(t, 200, conf.NATSPendingMaxMB)
	assert.Equal(t, 1048576, conf.ListenerBatchBytes)
	assert.Equal(t, 600, conf.MaxTimeDeltaSecs)
	assert.Equal(t, false, conf.Debug)
	assert.Len(t, conf.Rule, 0)
}

func TestDefaultPortListener(t *testing.T) {
	conf, err := parseConfig(`mode = "listener"`)
	require.NoError(t, err)
	assert.Equal(t, 10001, conf.Port)
}

func TestDefaultPortHTTPListener(t *testing.T) {
	conf, err := parseConfig(`mode = "listener_http"`)
	require.NoError(t, err)
	assert.Equal(t, 13337, conf.Port)
}

func TestNoMode(t *testing.T) {
	_, err := parseConfig("")
	assert.EqualError(t, err, "mode not specified in config")
}

func TestInvalidTOML(t *testing.T) {
	_, err := parseConfig("mode=\"writer\"\nbatch = abc")
	require.Error(t, err)
	assert.Regexp(t, ".+expected value but found.+", err.Error())
}

func TestRulesConfig(t *testing.T) {
	const rulesConfig = `
mode = "listener"
port = 10001

nats_address = "nats://localhost:4222"
nats_subject = ["spout"]
nats_subject_monitor = "spout-monitor"

influxdb_address = "localhost"
influxdb_port = 8086
influxdb_dbname = "junk_nats"

batch = 10
workers = 96

[[rule]]
type = "basic"
match = "hello"
subject = "hello-subject"

[[rule]]
type = "basic"
match = "world"
subject = "world-subject"
`
	conf, err := parseConfig(rulesConfig)
	require.NoError(t, err, "config should be parsed")

	assert.Len(t, conf.Rule, 2)
	assert.Equal(t, conf.Rule[0], Rule{
		Rtype:   "basic",
		Match:   "hello",
		Subject: "hello-subject",
	})
	assert.Equal(t, conf.Rule[1], Rule{
		Rtype:   "basic",
		Match:   "world",
		Subject: "world-subject",
	})
}

func TestCommonOverlay(t *testing.T) {
	const commonConfig = `
batch = 50
influxdb_dbname = "massive"
`
	const specificConfig = `
mode = "listener"
batch = 100
debug = true
`
	Fs = afero.NewMemMapFs()
	afero.WriteFile(Fs, commonFileName, []byte(commonConfig), 0600)
	afero.WriteFile(Fs, testConfigFileName, []byte(specificConfig), 0600)

	conf, err := NewConfigFromFile(testConfigFileName)
	require.NoError(t, err)

	assert.Equal(t, "listener", conf.Mode)   // only set in specific config
	assert.Equal(t, 100, conf.BatchMessages) // overridden in specific config
	assert.Equal(t, "massive", conf.DBName)  // only set in common config
}

func TestInvalidTOMLInCommonConfig(t *testing.T) {
	const commonConfig = `
wat
`
	const specificConfig = `
mode = "listener"
batch = 100
debug = true
`

	Fs = afero.NewMemMapFs()
	afero.WriteFile(Fs, commonFileName, []byte(commonConfig), 0600)
	afero.WriteFile(Fs, testConfigFileName, []byte(specificConfig), 0600)

	_, err := NewConfigFromFile(testConfigFileName)
	require.Error(t, err)
	assert.Regexp(t, "/etc/influx-spout.toml: .+", err)
}

type failingOpenFs struct{ afero.Fs }

func (*failingOpenFs) Open(string) (afero.File, error) {
	return nil, errors.New("boom")
}

func TestErrorOpeningCommonFile(t *testing.T) {
	Fs = new(failingOpenFs)

	_, err := NewConfigFromFile(testConfigFileName)
	assert.EqualError(t, err, "boom")
}

func TestOpenError(t *testing.T) {
	Fs = afero.NewMemMapFs()

	conf, err := NewConfigFromFile("/does/not/exist")
	assert.Nil(t, conf)
	assert.Error(t, err)
}

func parseConfig(content string) (*Config, error) {
	Fs = afero.NewMemMapFs()
	afero.WriteFile(Fs, testConfigFileName, []byte(content), 0600)

	return NewConfigFromFile(testConfigFileName)
}
