// Copyright (c) 2015-2026 libreFS contributors
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

// ── helpers ──────────────────────────────────────────────────────────────────

// makeDiffCh feeds two slices of ClientContent into difference() and collects
// all results.
func makeDiffCh(srcItems, tgtItems []*ClientContent, opts mirrorOptions) []diffMessage {
	srcCh := make(chan *ClientContent, len(srcItems)+1)
	tgtCh := make(chan *ClientContent, len(tgtItems)+1)
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

// makeDiffChSimilar is like makeDiffCh but passes returnSimilar=true so
// differInNone entries are also emitted.
func makeDiffChSimilar(srcItems, tgtItems []*ClientContent, opts mirrorOptions) []diffMessage {
	srcCh := make(chan *ClientContent, len(srcItems)+1)
	tgtCh := make(chan *ClientContent, len(tgtItems)+1)
	for _, c := range srcItems {
		srcCh <- c
	}
	for _, c := range tgtItems {
		tgtCh <- c
	}
	close(srcCh)
	close(tgtCh)

	var msgs []diffMessage
	for msg := range difference("s3://src/", srcCh, "s3://tgt/", tgtCh, opts, true) {
		msgs = append(msgs, msg)
	}
	return msgs
}

// reg returns a regular-file ClientContent with the given URL, size, ETag and
// modification time.
func reg(url string, size int64, etag string, modtime time.Time) *ClientContent {
	return &ClientContent{
		URL:  *newClientURL(url),
		Size: size,
		Type: os.FileMode(0o644),
		ETag: etag,
		Time: modtime,
	}
}

// regMeta is like reg but with extra user metadata.
func regMeta(url string, size int64, meta map[string]string) *ClientContent {
	c := reg(url, size, "", time.Now())
	c.UserMetadata = meta
	return c
}

// dir returns a directory-type ClientContent.
func dir(url string) *ClientContent {
	return &ClientContent{
		URL:  *newClientURL(url),
		Type: os.ModeDir,
	}
}

// collectDiffTypes returns only the Diff field from each message.
func collectDiffTypes(msgs []diffMessage) []differType {
	out := make([]differType, len(msgs))
	for i, m := range msgs {
		out[i] = m.Diff
	}
	return out
}

// ── differType.String ─────────────────────────────────────────────────────────

func TestDifferTypeString(t *testing.T) {
	cases := []struct {
		d    differType
		want string
	}{
		{differInNone, ""},
		{differInSize, "size"},
		{differInMetadata, "metadata"},
		{differInAASourceMTime, "mm-source-mtime"},
		{differInETag, "etag"},
		{differInType, "type"},
		{differInFirst, "only-in-first"},
		{differInSecond, "only-in-second"},
		{differInUnknown, "unknown"},
		{differType(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("differType(%d).String() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

// ── getSourceModTimeKey ───────────────────────────────────────────────────────

func TestGetSourceModTimeKey(t *testing.T) {
	ts := "2026-04-18T12:00:00Z"

	cases := []struct {
		name string
		meta map[string]string
		want string
	}{
		{
			name: "canonical key",
			meta: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": ts},
			want: ts,
		},
		{
			name: "lowercase canonical key",
			meta: map[string]string{"x-amz-meta-mm-source-mtime": ts},
			want: ts,
		},
		{
			name: "short key lowercase",
			meta: map[string]string{"mm-source-mtime": ts},
			want: ts,
		},
		{
			name: "short key mixed case",
			meta: map[string]string{"Mm-Source-Mtime": ts},
			want: ts,
		},
		{
			name: "no key present",
			meta: map[string]string{"other-key": "val"},
			want: "",
		},
		{
			name: "nil map",
			meta: nil,
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := getSourceModTimeKey(tc.meta)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── metadataEqual ─────────────────────────────────────────────────────────────

func TestMetadataEqual(t *testing.T) {
	ts := "2026-04-18T12:00:00Z"

	cases := []struct {
		name  string
		m1    map[string]string
		m2    map[string]string
		equal bool
	}{
		{
			name:  "both nil",
			m1:    nil,
			m2:    nil,
			equal: true,
		},
		{
			name:  "equal maps",
			m1:    map[string]string{"Content-Type": "text/plain"},
			m2:    map[string]string{"Content-Type": "text/plain"},
			equal: true,
		},
		{
			name:  "different values",
			m1:    map[string]string{"Content-Type": "text/plain"},
			m2:    map[string]string{"Content-Type": "application/json"},
			equal: false,
		},
		{
			name:  "extra key in m1",
			m1:    map[string]string{"Content-Type": "text/plain", "X-Custom": "a"},
			m2:    map[string]string{"Content-Type": "text/plain"},
			equal: false,
		},
		{
			name:  "extra key in m2",
			m1:    map[string]string{"Content-Type": "text/plain"},
			m2:    map[string]string{"Content-Type": "text/plain", "X-Custom": "a"},
			equal: false,
		},
		{
			name: "active-active key ignored",
			m1:   map[string]string{"X-Amz-Meta-Mm-Source-Mtime": ts, "Content-Type": "text/plain"},
			m2:   map[string]string{"Content-Type": "text/plain"},
			// the aa key is skipped by metadataEqual
			equal: true,
		},
		{
			name: "lowercase active-active key ignored",
			m1:   map[string]string{"x-amz-meta-mm-source-mtime": ts},
			m2:   map[string]string{},
			equal: true,
		},
		{
			name:  "empty maps",
			m1:    map[string]string{},
			m2:    map[string]string{},
			equal: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := metadataEqual(tc.m1, tc.m2)
			if got != tc.equal {
				t.Errorf("metadataEqual() = %v, want %v", got, tc.equal)
			}
		})
	}
}

// ── activeActiveModTimeUpdated ────────────────────────────────────────────────

func TestActiveActiveModTimeUpdated(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name string
		src  *ClientContent
		dst  *ClientContent
		want bool
	}{
		{
			name: "nil src",
			src:  nil,
			dst:  reg("s3://tgt/f", 100, "", now),
			want: false,
		},
		{
			name: "nil dst",
			src:  reg("s3://src/f", 100, "", now),
			dst:  nil,
			want: false,
		},
		{
			name: "both nil",
			want: false,
		},
		{
			name: "zero src time",
			src:  reg("s3://src/f", 100, "", time.Time{}),
			dst:  reg("s3://tgt/f", 100, "", now),
			want: false,
		},
		{
			name: "zero dst time",
			src:  reg("s3://src/f", 100, "", now),
			dst:  reg("s3://tgt/f", 100, "", time.Time{}),
			want: false,
		},
		{
			name: "src newer than dst (no metadata)",
			src:  reg("s3://src/f", 100, "", now.Add(time.Hour)),
			dst:  reg("s3://tgt/f", 100, "", now),
			want: true,
		},
		{
			name: "dst newer than src (no metadata)",
			src:  reg("s3://src/f", 100, "", now),
			dst:  reg("s3://tgt/f", 100, "", now.Add(time.Hour)),
			want: false,
		},
		{
			name: "equal times (no metadata)",
			src:  reg("s3://src/f", 100, "", now),
			dst:  reg("s3://tgt/f", 100, "", now),
			want: false,
		},
		{
			name: "src origin modtime newer via metadata",
			src: &ClientContent{
				URL:          *newClientURL("s3://src/f"),
				Type:         os.FileMode(0o644),
				Time:         now,
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": now.Add(2 * time.Hour).Format(time.RFC3339Nano)},
			},
			dst: &ClientContent{
				URL:          *newClientURL("s3://tgt/f"),
				Type:         os.FileMode(0o644),
				Time:         now.Add(time.Hour),
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": now.Add(time.Hour).Format(time.RFC3339Nano)},
			},
			want: true,
		},
		{
			name: "dst origin modtime newer via metadata",
			src: &ClientContent{
				URL:          *newClientURL("s3://src/f"),
				Type:         os.FileMode(0o644),
				Time:         now,
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": now.Format(time.RFC3339Nano)},
			},
			dst: &ClientContent{
				URL:          *newClientURL("s3://tgt/f"),
				Type:         os.FileMode(0o644),
				Time:         now,
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": now.Add(time.Hour).Format(time.RFC3339Nano)},
			},
			want: false,
		},
		{
			name: "invalid src metadata timestamp — ignored",
			src: &ClientContent{
				URL:          *newClientURL("s3://src/f"),
				Type:         os.FileMode(0o644),
				Time:         now.Add(time.Hour),
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": "not-a-date"},
			},
			dst: reg("s3://tgt/f", 100, "", now),
			want: false,
		},
		{
			name: "invalid dst metadata timestamp — ignored",
			src: reg("s3://src/f", 100, "", now.Add(time.Hour)),
			dst: &ClientContent{
				URL:          *newClientURL("s3://tgt/f"),
				Type:         os.FileMode(0o644),
				Time:         now,
				UserMetadata: map[string]string{"X-Amz-Meta-Mm-Source-Mtime": "not-a-date"},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := activeActiveModTimeUpdated(tc.src, tc.dst)
			if got != tc.want {
				t.Errorf("activeActiveModTimeUpdated() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ── differenceInternal via difference() ──────────────────────────────────────

func TestDifferenceOnlyInSource(t *testing.T) {
	now := time.Now()
	src := []*ClientContent{reg("s3://src/a.txt", 100, "aaa", now)}
	tgt := []*ClientContent{}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInFirst {
		t.Errorf("expected differInFirst, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceOnlyInTarget(t *testing.T) {
	now := time.Now()
	src := []*ClientContent{}
	tgt := []*ClientContent{reg("s3://tgt/b.txt", 100, "bbb", now)}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInSecond {
		t.Errorf("expected differInSecond, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceSizeMismatch(t *testing.T) {
	now := time.Now()
	src := []*ClientContent{reg("s3://src/f.bin", 100, "aaa", now)}
	tgt := []*ClientContent{reg("s3://tgt/f.bin", 200, "bbb", now)}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInSize {
		t.Errorf("expected differInSize, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceTypeMismatch(t *testing.T) {
	now := time.Now()
	// source is a regular file, target is a directory at the same path
	src := []*ClientContent{reg("s3://src/data", 100, "aaa", now)}
	tgt := []*ClientContent{dir("s3://tgt/data")}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInType {
		t.Errorf("expected differInType, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceMetadataMismatch(t *testing.T) {
	src := []*ClientContent{regMeta("s3://src/f.txt", 100, map[string]string{"Content-Type": "text/plain"})}
	tgt := []*ClientContent{regMeta("s3://tgt/f.txt", 100, map[string]string{"Content-Type": "application/json"})}

	msgs := makeDiffCh(src, tgt, mirrorOptions{isMetadata: true})
	if len(msgs) != 1 || msgs[0].Diff != differInMetadata {
		t.Errorf("expected differInMetadata, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceMetadataMatchNoSignal(t *testing.T) {
	// Same metadata — should produce no diff even with isMetadata=true.
	src := []*ClientContent{regMeta("s3://src/f.txt", 100, map[string]string{"Content-Type": "text/plain"})}
	tgt := []*ClientContent{regMeta("s3://tgt/f.txt", 100, map[string]string{"Content-Type": "text/plain"})}

	msgs := makeDiffCh(src, tgt, mirrorOptions{isMetadata: true})
	for _, m := range msgs {
		if m.Diff == differInMetadata {
			t.Error("expected no differInMetadata for equal metadata")
		}
	}
}

func TestDifferenceActiveActiveModTime(t *testing.T) {
	now := time.Now()
	src := []*ClientContent{{
		URL:          *newClientURL("s3://src/f.bin"),
		Type:         os.FileMode(0o644),
		Size:         100,
		Time:         now.Add(time.Hour),
		UserMetadata: map[string]string{},
	}}
	tgt := []*ClientContent{{
		URL:  *newClientURL("s3://tgt/f.bin"),
		Type: os.FileMode(0o644),
		Size: 100,
		Time: now,
	}}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInAASourceMTime {
		t.Errorf("expected differInAASourceMTime, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceAlphabeticOrdering(t *testing.T) {
	// "a.txt" < "b.txt" alphabetically.
	// src has a.txt, tgt has b.txt → a.txt is differInFirst, b.txt is differInSecond.
	now := time.Now()
	src := []*ClientContent{reg("s3://src/a.txt", 100, "aaa", now)}
	tgt := []*ClientContent{reg("s3://tgt/b.txt", 100, "bbb", now)}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	types := collectDiffTypes(msgs)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d: %v", len(msgs), types)
	}
	if msgs[0].Diff != differInFirst {
		t.Errorf("msg[0]: expected differInFirst, got %v", msgs[0].Diff)
	}
	if msgs[1].Diff != differInSecond {
		t.Errorf("msg[1]: expected differInSecond, got %v", msgs[1].Diff)
	}
}

func TestDifferenceNoDiff(t *testing.T) {
	// Same path, same size, same ETag, same time — no diff emitted without returnSimilar.
	now := time.Now()
	src := []*ClientContent{reg("s3://src/f.txt", 100, "abc", now)}
	tgt := []*ClientContent{reg("s3://tgt/f.txt", 100, "abc", now)}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	if len(msgs) != 0 {
		t.Errorf("expected no diff messages, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceReturnSimilar(t *testing.T) {
	// With returnSimilar=true, identical files should emit differInNone.
	now := time.Now()
	src := []*ClientContent{reg("s3://src/f.txt", 100, "abc", now)}
	tgt := []*ClientContent{reg("s3://tgt/f.txt", 100, "abc", now)}

	msgs := makeDiffChSimilar(src, tgt, mirrorOptions{})
	if len(msgs) != 1 || msgs[0].Diff != differInNone {
		t.Errorf("expected differInNone with returnSimilar, got %v", collectDiffTypes(msgs))
	}
}

func TestDifferenceMultipleFiles(t *testing.T) {
	// src: a.txt, c.txt   tgt: b.txt
	// a.txt → differInFirst, b.txt → differInSecond, c.txt → differInFirst
	now := time.Now()
	src := []*ClientContent{
		reg("s3://src/a.txt", 100, "aaa", now),
		reg("s3://src/c.txt", 100, "ccc", now),
	}
	tgt := []*ClientContent{
		reg("s3://tgt/b.txt", 100, "bbb", now),
	}

	msgs := makeDiffCh(src, tgt, mirrorOptions{})
	types := collectDiffTypes(msgs)
	expected := []differType{differInFirst, differInSecond, differInFirst}
	if len(types) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, types)
	}
	for i := range expected {
		if types[i] != expected[i] {
			t.Errorf("msg[%d]: want %v, got %v", i, expected[i], types[i])
		}
	}
}

func TestDifferenceEmptyBoth(t *testing.T) {
	msgs := makeDiffCh([]*ClientContent{}, []*ClientContent{}, mirrorOptions{})
	if len(msgs) != 0 {
		t.Errorf("expected no messages for empty source and target, got %d", len(msgs))
	}
}

// ── --md5 ETag comparison ─────────────────────────────────────────────────────

func TestDifferenceETag(t *testing.T) {
	now := time.Now()
	// Target is newer (simulates upload timestamp being after source modtime —
	// the exact scenario where activeActiveModTimeUpdated returns false).
	tgtTime := now.Add(10 * time.Minute)

	src := []*ClientContent{reg("s3://src/data/file.bin", 1024, "abc123", now)}
	tgt := []*ClientContent{reg("s3://tgt/data/file.bin", 1024, "def456", tgtTime)}

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
		src2 := []*ClientContent{reg("s3://src/data/file.bin", 1024, "abc123", now)}
		tgt2 := []*ClientContent{reg("s3://tgt/data/file.bin", 1024, "abc123", tgtTime)}
		msgs := makeDiffCh(src2, tgt2, mirrorOptions{md5: true})
		for _, m := range msgs {
			if m.Diff == differInETag || m.Diff == differInSize {
				t.Errorf("identical ETags should not trigger a copy, got diff=%v", m.Diff)
			}
		}
	})

	t.Run("empty_etag_skips_comparison", func(t *testing.T) {
		// If either ETag is empty (e.g. filesystem source), skip ETag check.
		src3 := []*ClientContent{reg("s3://src/f", 1024, "", now)}
		tgt3 := []*ClientContent{reg("s3://tgt/f", 1024, "def456", tgtTime)}
		msgs := makeDiffCh(src3, tgt3, mirrorOptions{md5: true})
		for _, m := range msgs {
			if m.Diff == differInETag {
				t.Error("empty src ETag should not trigger differInETag")
			}
		}
	})
}

// ── matchExcludeOptions (existing, kept for completeness) ────────────────────

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

// ── matchExcludeBucketOptions ─────────────────────────────────────────────────

func TestMatchExcludeBucketOptions(t *testing.T) {
	cases := []struct {
		name    string
		buckets []string
		suffix  string
		match   bool
	}{
		{
			name:    "no patterns — no match",
			buckets: nil,
			suffix:  "mybucket/path/to/file",
			match:   false,
		},
		{
			name:    "exact bucket match",
			buckets: []string{"mybucket"},
			suffix:  "mybucket/path/to/file",
			match:   true,
		},
		{
			name:    "wildcard bucket match",
			buckets: []string{"my*"},
			suffix:  "mybucket/path/to/file",
			match:   true,
		},
		{
			name:    "no match different bucket",
			buckets: []string{"other"},
			suffix:  "mybucket/path/to/file",
			match:   false,
		},
		{
			name:    "leading slash stripped",
			buckets: []string{"mybucket"},
			suffix:  "/mybucket/path/to/file",
			match:   true,
		},
		{
			name:    "multiple patterns first matches",
			buckets: []string{"other", "my*"},
			suffix:  "mybucket/key",
			match:   true,
		},
		{
			name:    "multiple patterns none match",
			buckets: []string{"alpha", "beta"},
			suffix:  "mybucket/key",
			match:   false,
		},
		{
			name:    "empty suffix",
			buckets: []string{"mybucket"},
			suffix:  "",
			match:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchExcludeBucketOptions(tc.buckets, tc.suffix)
			if got != tc.match {
				t.Errorf("matchExcludeBucketOptions(%v, %q) = %v, want %v",
					tc.buckets, tc.suffix, got, tc.match)
			}
		})
	}
}
