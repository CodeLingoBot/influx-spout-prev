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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicRuleCreation(t *testing.T) {
	r := CreateBasicRule("hello", "hello-subject")
	assert.Equal(t, "hello-subject", r.subject, "subject must match")
}

func TestBasicRule(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateBasicRule("hello", ""))

	assert.Equal(t, 0, rs.Lookup([]byte("hello,a=b x=y")))
	assert.Equal(t, 0, rs.Lookup([]byte("hello a=b,x=y")))

	assert.Equal(t, -1, rs.Lookup([]byte("cocacola a=b x=y")))
	assert.Equal(t, -1, rs.Lookup([]byte("pepsi,a=b x=y")))

	// Should only match the measurement name.
	assert.Equal(t, -1, rs.Lookup([]byte("pepsi,hello=b x=y")))
	assert.Equal(t, -1, rs.Lookup([]byte("pepsi,a=b hello=y")))
}

func TestBasicRuleUnescapes(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateBasicRule("hell o", ""))

	assert.Equal(t, 0, rs.Lookup([]byte(`hell\ o foo=bar`)))
	assert.Equal(t, 0, rs.Lookup([]byte(`hell\ o,foo=bar`)))
	assert.Equal(t, -1, rs.Lookup([]byte(`hell o foo=bar`)))
	assert.Equal(t, -1, rs.Lookup([]byte(`hell\,o foo=bar`)))
}

func TestRegexRule(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateRegexRule("(^hel|,etc=false)", ""))

	assert.Equal(t, 0, rs.Lookup(
		[]byte("hello,host=gopher01 somefield=11,etc=false")))
	assert.Equal(t, 0, rs.Lookup(
		[]byte("bye,host=gopher01 somefield=11,etc=false")))

	assert.Equal(t, -1, rs.Lookup(
		[]byte("cocacola,host=gopher01 somefield=11,etc=true")))
	assert.Equal(t, -1, rs.Lookup(
		[]byte("pepsi host=gopher01,somefield=11,etc=true")))
}

func TestRegexRuleUnescapes(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateRegexRule("hell +o", ""))

	assert.Equal(t, 0, rs.Lookup([]byte(`hell\ o x=y`)))
	assert.Equal(t, 0, rs.Lookup([]byte(`hell\ \ oworld x=y`)))
	assert.Equal(t, 0, rs.Lookup([]byte(`foo hell\ o=y`)))
	assert.Equal(t, -1, rs.Lookup([]byte(`hell x=1,x=y`)))
}

func TestNegativeRegexRule(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateNegativeRegexRule("hel|low", ""))

	assert.Equal(t, -1, rs.Lookup([]byte("hello,host=gopher01 x=y")))
	assert.Equal(t, -1, rs.Lookup([]byte("bye,host=gopher01 x=low")))

	assert.Equal(t, 0, rs.Lookup([]byte("HELLO,host=gopher01 x=y")))
	assert.Equal(t, 0, rs.Lookup([]byte("bye,host=gopher01 x=high")))
}

func TestNegativeRegexRuleUnescapes(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateNegativeRegexRule("hell +o", ""))

	assert.Equal(t, -1, rs.Lookup([]byte(`hell\ o,host=gopher01 x=y`)))
	assert.Equal(t, -1, rs.Lookup([]byte(`bye,host=gopher01 x=hell\ \ o`)))

	assert.Equal(t, 0, rs.Lookup([]byte("HELLO,host=gopher01 x=y")))
	assert.Equal(t, 0, rs.Lookup([]byte("bye,host=gopher01 x=hello")))
}

func TestMultipleRules(t *testing.T) {
	rs := new(RuleSet)
	rs.Append(CreateBasicRule("hello", "a"))
	rs.Append(CreateRegexRule(".+ing", "b"))
	rs.Append(CreateNegativeRegexRule("foo", "c"))

	assert.Equal(t, 3, rs.Count())
	assert.Equal(t, []string{"a", "b", "c"}, rs.Subjects())

	assert.Equal(t, 0, rs.Lookup([]byte("hello,host=gopher01")))
	assert.Equal(t, 1, rs.Lookup([]byte("singing,host=gopher01")))
	assert.Equal(t, 2, rs.Lookup([]byte("bar,host=gopher01")))
	assert.Equal(t, -1, rs.Lookup([]byte("foo,host=gopher01")))
}

func TestMeasurementName(t *testing.T) {
	check := func(input, expected string) {
		assert.Equal(t, expected, string(measurementName([]byte(input))),
			"measurementName(%q)", input)
	}

	check(``, ``)
	check(`h`, `h`)
	check("日", "日")
	check(`hello`, `hello`)
	check("日本語", "日本語")
	check(` `, ``)
	check(`,`, ``)
	check(`h world`, `h`)
	check(`h,world`, `h`)
	check(`hello world`, `hello`)
	check(`hello,world`, `hello`)
	check(`hello\ world`, `hello\ world`)
	check(`hello\,world`, `hello\,world`)
	check(`hello\ world more`, `hello\ world`)
	check(`hello\,world,more`, `hello\,world`)
	check(`hello\ 日本語 more`, `hello\ 日本語`)
	check(`hello\,日本語 more`, `hello\,日本語`)
	check(`日本語\ hello more`, `日本語\ hello`)
	check(`日本語\,hello more`, `日本語\,hello`)
	check(`\ `, `\ `)
	check(`\,`, `\,`)
	check(`\`, `\`)
	check(`h\`, `h\`)
	check(`hello\`, `hello\`)
}

var result int

func BenchmarkLineLookup(b *testing.B) {
	rs := new(RuleSet)
	rs.Append(CreateBasicRule("hello", ""))
	line := []byte("hello world=42")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = rs.Lookup(line)
	}
}

func BenchmarkLineLookupRegex(b *testing.B) {
	rs := new(RuleSet)
	rs.Append(CreateRegexRule("hello|abcde", ""))
	line := []byte("hello world=42")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = rs.Lookup(line)
	}
}

func BenchmarkLineLookupNegativeRegex(b *testing.B) {
	rs := new(RuleSet)
	rs.Append(CreateNegativeRegexRule("hello|abcde", ""))
	line := []byte("hello world=42")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = rs.Lookup(line)
	}
}

func BenchmarkProcessBatch(b *testing.B) {
	// Run the Filter worker with a fake NATS connection.
	rs := new(RuleSet)
	rs.Append(CreateBasicRule("hello", "hello-out"))
	rs.Append(CreateRegexRule("foo|bar", "foobar-out"))

	w, err := newWorker(600, rs, initStats(rs), false, nullNATSConnect, "junk")
	require.NoError(b, err)

	lines := []string{
		"hello,host=gopher01 somefield=11,etc=false",
		"bar,host=gopher02 somefield=14",
		"pepsi host=gopher01,cheese=stilton",
		"hello,host=gopher01 somefield=11,etc=false",
		"bar,host=gopher02 somefield=14",
		"pepsi host=gopher01,cheese=stilton",
		"hello,host=gopher01 somefield=11,etc=false",
		"bar,host=gopher02 somefield=14",
		"pepsi host=gopher01,cheese=stilton",
	}

	// Add a timestamp to each line.
	ts := strconv.FormatInt(time.Now().UnixNano(), 10)
	for i, line := range lines {
		lines[i] = line + " " + ts
	}

	batch := []byte(strings.Join(lines, "\n"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.processBatch(batch)
	}
}

func nullNATSConnect() (natsConn, error) {
	return new(nullConn), nil
}

type nullConn struct {
	natsConn
}

func (*nullConn) Publish(string, []byte) error {
	return nil
}
