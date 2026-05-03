// Copyright (c) 2015-2022 libreFS, Inc.
//
// This file is part of libreFS Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	minio "github.com/libreFS/librefs-go/v7"
)

func TestLineTrunc(t *testing.T) {
	cases := []struct {
		content string
		maxLen  int
		want    string
	}{
		// shorter than limit → unchanged
		{"hello", 10, "hello"},
		// exactly the limit → unchanged
		{"hello", 5, "hello"},
		// longer → truncated with ellipsis in middle
		{"abcdefghij", 6, "abc…hij"},
		// unicode runes handled correctly (each rune counts as 1, not byte-count)
		{"αβγδεζηθ", 4, "αβ…ηθ"},
	}
	for _, c := range cases {
		got := lineTrunc(c.content, c.maxLen)
		if got != c.want {
			t.Errorf("lineTrunc(%q, %d) = %q, want %q", c.content, c.maxLen, got, c.want)
		}
	}
}

func TestIsURLContains(t *testing.T) {
	cases := []struct {
		src  string
		tgt  string
		sep  string
		want bool
	}{
		// target is a sub-path of source
		{"alias/bucket", "alias/bucket/dir", "/", true},
		// target equals source (after sep appended)
		{"alias/bucket", "alias/bucket", "/", true},
		// target is unrelated
		{"alias/bucket", "alias/other", "/", false},
		// source already has trailing sep
		{"alias/bucket/", "alias/bucket/dir/", "/", true},
		// completely different
		{"s3://a/b", "s3://c/d", "/", false},
	}
	for _, c := range cases {
		got := isURLContains(c.src, c.tgt, c.sep)
		if got != c.want {
			t.Errorf("isURLContains(%q, %q, %q) = %v, want %v", c.src, c.tgt, c.sep, got, c.want)
		}
	}
}

func TestConservativeFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello world", "hello_world"},
		{"file-name_v2.0", "file-name_v2_0"},
		{"(test)[value]", "(test)[value]"},
		{"path/to/file", "path_to_file"},
		// leading/trailing underscores (from replacements) are trimmed
		{"/leading", "leading"},
		{"trailing/", "trailing"},
		// allowed special chars pass through
		{"file+name%20", "file+name%20"},
		// only underscore replacements get trimmed, not interior ones
		{"a_b", "a_b"},
	}
	for _, c := range cases {
		got := conservativeFileName(c.in)
		if got != c.want {
			t.Errorf("conservativeFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetLookupType(t *testing.T) {
	cases := []struct {
		in   string
		want minio.BucketLookupType
	}{
		{"off", minio.BucketLookupDNS},
		{"OFF", minio.BucketLookupDNS},
		{"on", minio.BucketLookupPath},
		{"ON", minio.BucketLookupPath},
		{"auto", minio.BucketLookupAuto},
		{"", minio.BucketLookupAuto},
		{"anything-else", minio.BucketLookupAuto},
	}
	for _, c := range cases {
		got := getLookupType(c.in)
		if got != c.want {
			t.Errorf("getLookupType(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCenterText(t *testing.T) {
	cases := []struct {
		s    string
		w    int
		want int // expected total rune width of result
	}{
		{"hi", 10, 10},
		{"hello", 5, 5},
		{"x", 1, 1},
	}
	for _, c := range cases {
		got := centerText(c.s, c.w)
		// strip ANSI (none in our inputs) and check length
		stripped := strings.TrimSpace(got)
		if stripped != c.s {
			t.Errorf("centerText(%q, %d): core text = %q, want %q", c.s, c.w, stripped, c.s)
		}
		if len([]rune(got)) != c.want {
			t.Errorf("centerText(%q, %d): total width = %d, want %d", c.s, c.w, len([]rune(got)), c.want)
		}
	}
}

func TestIsOlderNewer(t *testing.T) {
	// empty ref → always false
	now := time.Now()
	if isOlder(now, "") {
		t.Error("isOlder(now, \"\") should be false")
	}
	if isNewer(now, "") {
		t.Error("isNewer(now, \"\") should be false")
	}

	// object created 2 days ago, threshold 1 day
	// age (2d) > threshold (1d) → isOlder should return false (age < threshold is false)
	// isNewer age(2d) >= threshold(1d) → true
	old := now.Add(-48 * time.Hour)
	if isOlder(old, "1d") {
		t.Error("isOlder(2-days-ago, \"1d\"): expected false (age 2d is not < 1d threshold)")
	}
	if !isNewer(old, "1d") {
		t.Error("isNewer(2-days-ago, \"1d\"): expected true (age 2d >= 1d threshold)")
	}

	// object created 12 hours ago, threshold 1 day
	// age (12h) < threshold (1d) → isOlder true, isNewer false
	recent := now.Add(-12 * time.Hour)
	if !isOlder(recent, "1d") {
		t.Error("isOlder(12h-ago, \"1d\"): expected true (age 12h < 1d threshold)")
	}
	if isNewer(recent, "1d") {
		t.Error("isNewer(12h-ago, \"1d\"): expected false (age 12h < 1d threshold)")
	}
}

func TestParseAttribute(t *testing.T) {
	metaDataCases := []struct {
		input  string
		output map[string]string
		err    error
		status bool
	}{
		// // When blank value is passed.
		{"", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		//  When space is passed.
		{"  ", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		// When / is passed.
		{"/", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		// When "atime:" is passed.
		{"atime:/", map[string]string{"atime": ""}, ErrInvalidFileSystemAttribute, false},
		// When "atime:" is passed.
		{"atime", map[string]string{"atime": ""}, nil, true},
		//  When "atime:" is passed.
		{"atime:", map[string]string{"atime": ""}, nil, true},
		// Passing a valid value
		{
			"atime:1/gid:1/gname:a/md:/mode:3/mtime:1/uid:1/uname:a",
			map[string]string{
				"atime": "1",
				"gid":   "1",
				"gname": "a",
				"md":    "",
				"mode":  "3",
				"mtime": "1",
				"uid":   "1",
				"uname": "a",
			},
			nil, true,
		},
	}

	for idx, testCase := range metaDataCases {
		meta, err := parseAttribute(map[string]string{
			metadataKey: testCase.input,
		})
		if testCase.status == true {
			if err != nil {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, err)
			}
			if !reflect.DeepEqual(meta, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, meta)
			}
		}
		if testCase.status == false {
			if !reflect.DeepEqual(meta, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, meta)
			}
			if err != testCase.err {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, err)
			}
		}

	}
}
