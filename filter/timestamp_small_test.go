// Copyright 2018 Jump Trading
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

package filter

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTimestamp(t *testing.T) {
	ts := time.Date(1997, 6, 5, 4, 3, 2, 1, time.UTC).UnixNano()
	tsStr := strconv.FormatInt(ts, 10)
	defaultTs := time.Now().UnixNano()

	check := func(input string, expected int64) {
		assert.Equal(
			t, expected,
			extractTimestamp([]byte(input), defaultTs),
			"extractTimestamp(%q)", input)
	}

	check("", defaultTs)
	check(" ", defaultTs)
	check("weather temp=99", defaultTs)
	check("weather,city=paris temp=60", defaultTs)
	check("weather,city=paris temp=99,humidity=100", defaultTs)
	check("weather temp=99 "+tsStr, ts)
	check("weather temp=99 "+tsStr+"\n", ts)
	check("weather,city=paris temp=60 "+tsStr, ts)
	check("weather,city=paris temp=60,humidity=100 "+tsStr, ts)
	check("weather,city=paris temp=60,humidity=100 "+tsStr+"\n", ts)

	// Various invalid timestamps
	check("weather temp=99 "+tsStr+" ", defaultTs)
	check("weather temp=99 xxxxxxxxxxxxxxxxxxx", defaultTs)
	check("weather temp=99 152076148x803180202", defaultTs)  // non-digit embedded
	check("weather temp=99 11520761485803180202", defaultTs) // too long
	check("weather temp=99 -"+tsStr, defaultTs)
	check(tsStr, defaultTs)
}

func TestFastParseInt(t *testing.T) {
	check := func(input string, expected int64) {
		actual, err := fastParseInt([]byte(input))
		require.NoError(t, err)
		assert.Equal(t, expected, actual, "fastParseInt(%q)", input)
	}

	shouldFail := func(input string) {
		_, err := fastParseInt([]byte(input))
		assert.Error(t, err)
	}

	check("0", 0)
	check("1", 1)
	check("9", 9)
	check("10", 10)
	check("99", 99)
	check("101", 101)
	check("9223372036854775807", (1<<63)-1) // max int64 value

	shouldFail("9223372036854775808") // max int64 value + 1
	shouldFail("-1")                  // negatives not supported
	shouldFail("x")
}
