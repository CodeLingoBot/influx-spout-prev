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

package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
)

// The file here is parsed first. The config file given on the command
// line is then overlaid on top of it.
const commonFileName = "/etc/influx-spout.toml"

type RawRule struct {
	Rtype   string `toml:"type"`
	Match   string `toml:"match"`
	Channel string `toml:"channel"`
}

// Config replaces our configuration values
type Config struct {
	NATSAddress       string    `toml:"nats_address"`
	NATSTopic         []string  `toml:"nats_topic"`
	NATSTopicMonitor  string    `toml:"nats_topic_monitor"`
	NATSTopicJunkyard string    `toml:"nats_topic_junkyard"`
	InfluxDBAddress   string    `toml:"influxdb_address"`
	InfluxDBPort      int       `toml:"influxdb_port"`
	DBName            string    `toml:"influxdb_dbname"`
	BatchMessages     int       `toml:"batch"`
	IsTesting         bool      `toml:"testing_mode"`
	Port              int       `toml:"port"`
	Mode              string    `toml:"mode"`
	WriterWorkers     int       `toml:"workers"`
	WriteTimeoutSecs  int       `toml:"write_timeout_secs"`
	NATSPendingMaxMB  int       `toml:"nats_pending_max_mb"`
	Rule              []RawRule `toml:"rule"`
	Debug             bool      `toml:"debug"`
}

func newDefaultConfig() *Config {
	return &Config{
		NATSAddress:       "nats://localhost:4222",
		NATSTopic:         []string{"influx-spout"},
		NATSTopicMonitor:  "influx-spout-monitor",
		NATSTopicJunkyard: "influx-spout-junk",
		InfluxDBAddress:   "localhost",
		InfluxDBPort:      8086,
		DBName:            "influx-spout-junk",
		BatchMessages:     10,
		WriterWorkers:     10,
		WriteTimeoutSecs:  30,
		NATSPendingMaxMB:  200,
	}
}

// NewConfig parses the specified configuration file and returns a
// Config.
func NewConfigFromFile(fileName string) (*Config, error) {
	conf := newDefaultConfig()
	if err := readConfig(commonFileName, conf); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := readConfig(fileName, conf); err != nil {
		return nil, err
	}

	if conf.Mode == "" {
		return nil, errors.New("mode not specified in config")
	}

	// Set dynamic defaults.
	if conf.Mode == "listener" && conf.Port == 0 {
		conf.Port = 10001
	} else if conf.Mode == "listener_http" && conf.Port == 0 {
		conf.Port = 13337
	}
	return conf, nil
}

func readConfig(fileName string, conf *Config) error {
	f, err := fs.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = toml.DecodeReader(f, conf)
	if err != nil {
		return fmt.Errorf("%s: %v", fileName, err)
	}
	return nil
}

var fs = afero.NewOsFs()