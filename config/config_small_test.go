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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfigFileName = "config.toml"

func TestCorrectConfigFile(t *testing.T) {
	const validConfigSample = `
testing_mode = false

mode = "listener"
port = 10001

nats_address = "nats://localhost:4222"
nats_topic = ["spout"]
nats_topic_monitor = "spout-monitor"

influxdb_address = "localhost"
influxdb_port = 8086
influxdb_dbname = "junk_nats"

batch = 10
workers = 96

write_timeout_secs = 32
nats_pending_max_mb = 100
`
	conf, err := parseConfig(validConfigSample)
	require.NoError(t, err, "Couldn't parse a valid config: %v\n", err)

	assert.Equal(t, "listener", conf.Mode, "Mode must match")
	assert.Equal(t, 10001, conf.Port, "Port must match")
	assert.Equal(t, 10, conf.BatchMessages, "Batching must match")
	assert.Equal(t, 96, conf.WriterWorkers, "Workers must match")
	assert.Equal(t, 32, conf.WriteTimeoutSecs, "WriteTimeoutSecs must match")
	assert.Equal(t, 100, conf.NATSPendingMaxMB, "NATSPendingMaxMB must match")

	assert.Equal(t, 8086, conf.InfluxDBPort, "InfluxDB Port must match")
	assert.Equal(t, "junk_nats", conf.DBName, "InfluxDB DBname must match")
	assert.Equal(t, "localhost", conf.InfluxDBAddress, "InfluxDB address must match")

	assert.Equal(t, "spout", conf.NATSTopic[0], "Topic must match")
	assert.Equal(t, "spout-monitor", conf.NATSTopicMonitor, "Monitor topic must match")
	assert.Equal(t, "nats://localhost:4222", conf.NATSAddress, "Address must match")
}

func TestAllDefaults(t *testing.T) {
	conf, err := parseConfig(`mode = "writer"`)
	require.NoError(t, err)

	assert.Equal(t, "nats://localhost:4222", conf.NATSAddress)
	assert.Equal(t, []string{"influx-spout"}, conf.NATSTopic)
	assert.Equal(t, "influx-spout-monitor", conf.NATSTopicMonitor)
	assert.Equal(t, "influx-spout-junk", conf.NATSTopicJunkyard)
	assert.Equal(t, "localhost", conf.InfluxDBAddress)
	assert.Equal(t, 8086, conf.InfluxDBPort)
	assert.Equal(t, "influx-spout-junk", conf.DBName)
	assert.Equal(t, 10, conf.BatchMessages)
	assert.Equal(t, false, conf.IsTesting)
	assert.Equal(t, 0, conf.Port)
	assert.Equal(t, "writer", conf.Mode)
	assert.Equal(t, 10, conf.WriterWorkers)
	assert.Equal(t, 30, conf.WriteTimeoutSecs)
	assert.Equal(t, 200, conf.NATSPendingMaxMB)
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
testing_mode = false

mode = "listener"
port = 10001

nats_address = "nats://localhost:4222"
nats_topic = ["spout"]
nats_topic_monitor = "spout-monitor"

influxdb_address = "localhost"
influxdb_port = 8086
influxdb_dbname = "junk_nats"

batch = 10
workers = 96

[[rule]]
type = "basic"
match = "hello"
channel = "hello-chan"

[[rule]]
type = "basic"
match = "world"
channel = "world-chan"
`
	conf, err := parseConfig(rulesConfig)
	require.NoError(t, err, "config should be parsed")

	assert.Len(t, conf.Rule, 2)
	assert.Equal(t, conf.Rule[0], RawRule{
		Rtype:   "basic",
		Match:   "hello",
		Channel: "hello-chan",
	})
	assert.Equal(t, conf.Rule[1], RawRule{
		Rtype:   "basic",
		Match:   "world",
		Channel: "world-chan",
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
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, commonFileName, []byte(commonConfig), 0600)
	afero.WriteFile(fs, "config.toml", []byte(specificConfig), 0600)

	conf, err := NewConfigFromFile("config.toml")
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

	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, commonFileName, []byte(commonConfig), 0600)
	afero.WriteFile(fs, "config.toml", []byte(specificConfig), 0600)

	_, err := NewConfigFromFile("config.toml")
	require.Error(t, err)
	assert.Regexp(t, "/etc/influx-spout.toml: .+", err)
}

type failingOpenFs struct{ afero.Fs }

func (*failingOpenFs) Open(string) (afero.File, error) {
	return nil, errors.New("boom")
}

func TestErrorOpeningCommonFile(t *testing.T) {
	fs = new(failingOpenFs)

	_, err := NewConfigFromFile("config.toml")
	assert.EqualError(t, err, "boom")
}

func TestOpenError(t *testing.T) {
	fs = afero.NewMemMapFs()

	conf, err := NewConfigFromFile("/does/not/exist")
	assert.Nil(t, conf)
	assert.Error(t, err)
}

func parseConfig(content string) (*Config, error) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, testConfigFileName, []byte(content), 0600)

	return NewConfigFromFile(testConfigFileName)
}