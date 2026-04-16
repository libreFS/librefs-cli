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
	"os"
	"testing"
	"time"
)

var testCases = []struct {
	pattern []string

	srcSuffix string

	match bool

	typ ClientURLType
}{
	{nil, "testfile", false, objectStorage},
	{[]string{"test*"}, "testfile", true, objectStorage},
	{[]string{"file*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{"*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{""}, "file/abc/bcd/def", false, objectStorage},
	{[]string{"abc*"}, "file/abc/bcd/def", false, objectStorage},
	{[]string{"abc*", "*abc/*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{"*.txt"}, "file/abc/bcd/def.txt", true, objectStorage},
	{[]string{".*"}, ".sys", true, objectStorage},
	{[]string{"*."}, ".sys.", true, objectStorage},
	{nil, "testfile", false, fileSystem},
	{[]string{"test*"}, "testfile", true, fileSystem},
	{[]string{"file*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{"*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{""}, "file/abc/bcd/def", false, fileSystem},
	{[]string{"abc*"}, "file/abc/bcd/def", false, fileSystem},
	{[]string{"abc*", "*abc/*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{"abc*", "*abc/*"}, "/file/abc/bcd/def", true, fileSystem},
	{[]string{"*.txt"}, "file/abc/bcd/def.txt", true, fileSystem},
	{[]string{"*.txt"}, "/file/abc/bcd/def.txt", true, fileSystem},
	{[]string{".*"}, ".sys", true, fileSystem},
	{[]string{"*."}, ".sys.", true, fileSystem},
}

func TestExcludeOptions(t *testing.T) {
	for _, test := range testCases {
		if matchExcludeOptions(test.pattern, test.srcSuffix, test.typ) != test.match {
			t.Fatalf("Unexpected result %t, with pattern %s and srcSuffix %s \n", !test.match, test.pattern, test.srcSuffix)
		}
	}
}

// makeDiffCh feeds two slices of ClientContent into difference() and collects results.
func makeDiffCh(srcItems, tgtItems []*ClientContent, opts mirrorOptions) []diffMessage {
	srcCh := make(chan *ClientContent, len(srcItems))
	tgtCh := make(chan *ClientContent, len(tgtItems))
	for _, c := range srcItems {
		srcCh <- c
	}
	for _, c := range tgtItems {
		tgtCh <- c
	}
	close(srcCh)
	close(tgtCh)

	var msgs []diffMessage
	for msg := range difference("s3://src/", srcCh, "s3://tgt/", tgtCh, opts, false) {
		msgs = append(msgs, msg)
	}
	return msgs
}

func fileContent(url string, size int64, etag string, modtime time.Time) *ClientContent {
	return &ClientContent{
		URL:  *newClientURL(url),
		Size: size,
		Type: os.FileMode(0o644), // regular file
		ETag: etag,
		Time: modtime,
	}
}

// TestDifferenceETag verifies that when --md5 is set, two files with the same
// path and size but different ETags are reported as differInETag, not differInNone.
// This is the core bug: without the fix the file would be silently skipped.
func TestDifferenceETag(t *testing.T) {
	now := time.Now()
	// Target is newer (simulates upload timestamp being after source modtime —
	// the exact scenario where activeActiveModTimeUpdated returns false).
	tgtTime := now.Add(10 * time.Minute)

	src := []*ClientContent{fileContent("s3://src/data/file.bin", 1024, "abc123", now)}
	tgt := []*ClientContent{fileContent("s3://tgt/data/file.bin", 1024, "def456", tgtTime)}

	t.Run("md5_flag_detects_etag_diff", func(t *testing.T) {
		msgs := makeDiffCh(src, tgt, mirrorOptions{md5: true})
		if len(msgs) != 1 {
			t.Fatalf("expected 1 diff message, got %d", len(msgs))
		}
		if msgs[0].Diff != differInETag {
			t.Errorf("expected differInETag, got %v", msgs[0].Diff)
		}
	})

	t.Run("no_md5_flag_ignores_etag_diff", func(t *testing.T) {
		// Without --md5, same-size files with newer target timestamp should
		// not be copied (existing behavior preserved).
		msgs := makeDiffCh(src, tgt, mirrorOptions{md5: false})
		for _, m := range msgs {
			if m.Diff == differInETag {
				t.Error("differInETag should not be emitted when --md5 is not set")
			}
		}
	})

	t.Run("same_etag_not_copied", func(t *testing.T) {
		// Same ETag — should not trigger a copy even with --md5.
		src2 := []*ClientContent{fileContent("s3://src/data/file.bin", 1024, "abc123", now)}
		tgt2 := []*ClientContent{fileContent("s3://tgt/data/file.bin", 1024, "abc123", tgtTime)}
		msgs := makeDiffCh(src2, tgt2, mirrorOptions{md5: true})
		for _, m := range msgs {
			if m.Diff == differInETag || m.Diff == differInSize {
				t.Errorf("identical ETags should not trigger a copy, got diff=%v", m.Diff)
			}
		}
	})
}
