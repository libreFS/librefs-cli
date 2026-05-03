// Copyright (c) 2015-2023 libreFS, Inc.
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

package ilm

import (
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/libreFS/librefs-go/v7/pkg/lifecycle"
)

// ── ToILMRule ─────────────────────────────────────────────────────────────────

func TestToILMRule(t *testing.T) {
	// expiry days rule
	opts := LifecycleOptions{
		ID:         "test-rule",
		ExpiryDays: strPtr("30"),
	}
	rule, err := opts.ToILMRule()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.ID != "test-rule" {
		t.Errorf("ID = %q, want test-rule", rule.ID)
	}
	if rule.Status != "Enabled" {
		t.Errorf("Status = %q, want Enabled", rule.Status)
	}
	if rule.Expiration.Days != 30 {
		t.Errorf("Days = %d, want 30", rule.Expiration.Days)
	}

	// disabled status
	opts2 := LifecycleOptions{
		ExpiryDays: strPtr("7"),
		Status:     boolPtr(false),
	}
	rule2, err := opts2.ToILMRule()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule2.Status != "Disabled" {
		t.Errorf("Status = %q, want Disabled", rule2.Status)
	}

	// no action set → validation error
	emptyOpts := LifecycleOptions{ID: "empty"}
	if _, err := emptyOpts.ToILMRule(); err == nil {
		t.Error("expected error for rule with no action")
	}

	// invalid expiry days → error
	badOpts := LifecycleOptions{ExpiryDays: strPtr("notanumber")}
	if _, err := badOpts.ToILMRule(); err == nil {
		t.Error("expected error for invalid expiry days")
	}

	// noncurrent version expiry
	ndays := 14
	opts3 := LifecycleOptions{
		NoncurrentVersionExpirationDays: &ndays,
	}
	rule3, err := opts3.ToILMRule()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule3.NoncurrentVersionExpiration.NoncurrentDays != 14 {
		t.Errorf("NoncurrentDays = %d, want 14", rule3.NoncurrentVersionExpiration.NoncurrentDays)
	}
}

// ── RemoveILMRule ─────────────────────────────────────────────────────────────

func TestRemoveILMRule(t *testing.T) {
	cfg := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{ID: "rule-1", Status: "Enabled"},
			{ID: "rule-2", Status: "Enabled"},
		},
	}

	// remove existing rule
	result, err := RemoveILMRule(cfg, "rule-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rules) != 1 || result.Rules[0].ID != "rule-2" {
		t.Errorf("after remove, rules = %v, want [rule-2]", result.Rules)
	}

	// remove non-existent rule → error
	_, err = RemoveILMRule(cfg, "does-not-exist")
	if err == nil {
		t.Error("expected error removing non-existent rule ID")
	}

	// nil config → error
	_, err = RemoveILMRule(nil, "any")
	if err == nil {
		t.Error("expected error for nil config")
	}

	// empty rules → error
	_, err = RemoveILMRule(&lifecycle.Configuration{}, "any")
	if err == nil {
		t.Error("expected error for empty rules")
	}
}

// ── intPtr / boolPtr ──────────────────────────────────────────────────────────

func TestIntPtr(t *testing.T) {
	p := intPtr(42)
	if p == nil || *p != 42 {
		t.Errorf("intPtr(42) = %v, want pointer to 42", p)
	}
}

func TestBoolPtr(t *testing.T) {
	p := boolPtr(true)
	if p == nil || !*p {
		t.Errorf("boolPtr(true) = %v, want pointer to true", p)
	}
	p2 := boolPtr(false)
	if p2 == nil || *p2 {
		t.Errorf("boolPtr(false) = %v, want pointer to false", p2)
	}
}

// ── ApplyRuleFields ───────────────────────────────────────────────────────────

func TestApplyRuleFields(t *testing.T) {
	base := lifecycle.Rule{
		ID:     "test-rule",
		Status: "Enabled",
		Expiration: lifecycle.Expiration{
			Days: 30,
		},
	}

	// update expiry days
	opts := LifecycleOptions{ExpiryDays: strPtr("60")}
	dest := base
	if err := ApplyRuleFields(&dest, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Expiration.Days != 60 {
		t.Errorf("Days = %d, want 60", dest.Expiration.Days)
	}

	// update status to disabled
	opts2 := LifecycleOptions{Status: boolPtr(false)}
	dest2 := base
	if err := ApplyRuleFields(&dest2, opts2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest2.Status != "Disabled" {
		t.Errorf("Status = %q, want Disabled", dest2.Status)
	}

	// update storage class
	opts3 := LifecycleOptions{StorageClass: strPtr("GLACIER")}
	dest3 := base
	if err := ApplyRuleFields(&dest3, opts3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest3.Transition.StorageClass != "GLACIER" {
		t.Errorf("StorageClass = %q, want GLACIER", dest3.Transition.StorageClass)
	}

	// invalid expiry days → error propagated
	opts4 := LifecycleOptions{ExpiryDays: strPtr("notanumber")}
	dest4 := base
	if err := ApplyRuleFields(&dest4, opts4); err == nil {
		t.Error("expected error for invalid expiry days")
	}
}

func TestOptionFilter(t *testing.T) {
	emptyFilter := lifecycle.Filter{}
	emptyOpts := LifecycleOptions{}

	filterWithPrefix := lifecycle.Filter{
		Prefix: "doc/",
	}
	optsWithPrefix := LifecycleOptions{
		Prefix: strPtr("doc/"),
	}

	filterWithTag := lifecycle.Filter{
		Tag: lifecycle.Tag{
			Key:   "key1",
			Value: "value1",
		},
	}
	optsWithTag := LifecycleOptions{
		Tags: strPtr("key1=value1"),
	}

	filterWithSzLt := lifecycle.Filter{
		ObjectSizeLessThan: 100 * humanize.MiByte,
	}
	optsWithSzLt := LifecycleOptions{
		ObjectSizeLessThan: int64Ptr(100 * humanize.MiByte),
	}

	filterWithSzGt := lifecycle.Filter{
		ObjectSizeGreaterThan: 1 * humanize.MiByte,
	}
	optsWithSzGt := LifecycleOptions{
		ObjectSizeGreaterThan: int64Ptr(1 * humanize.MiByte),
	}

	filterWithAnd := lifecycle.Filter{
		And: lifecycle.And{
			Prefix: "doc/",
			Tags: []lifecycle.Tag{
				{
					Key:   "key1",
					Value: "value1",
				},
			},
			ObjectSizeLessThan:    100 * humanize.MiByte,
			ObjectSizeGreaterThan: 1 * humanize.MiByte,
		},
	}
	optsWithAnd := LifecycleOptions{
		Prefix:                strPtr("doc/"),
		Tags:                  strPtr("key1=value1"),
		ObjectSizeLessThan:    int64Ptr(100 * humanize.MiByte),
		ObjectSizeGreaterThan: int64Ptr(1 * humanize.MiByte),
	}

	tests := []struct {
		opts LifecycleOptions
		want lifecycle.Filter
	}{
		{
			opts: emptyOpts,
			want: emptyFilter,
		},
		{
			opts: optsWithPrefix,
			want: filterWithPrefix,
		},
		{
			opts: optsWithTag,
			want: filterWithTag,
		},
		{
			opts: optsWithSzGt,
			want: filterWithSzGt,
		},
		{
			opts: optsWithSzLt,
			want: filterWithSzLt,
		},
		{
			opts: optsWithAnd,
			want: filterWithAnd,
		},
	}

	filterEq := func(a, b lifecycle.Filter) bool {
		if a.ObjectSizeGreaterThan != b.ObjectSizeGreaterThan {
			return false
		}
		if a.ObjectSizeLessThan != b.ObjectSizeLessThan {
			return false
		}
		if a.Prefix != b.Prefix {
			return false
		}
		if a.Tag != b.Tag {
			return false
		}

		if a.And.ObjectSizeGreaterThan != b.And.ObjectSizeGreaterThan {
			return false
		}
		if a.And.ObjectSizeLessThan != b.And.ObjectSizeLessThan {
			return false
		}
		if a.And.Prefix != b.And.Prefix {
			return false
		}
		if len(a.And.Tags) != len(b.And.Tags) {
			return false
		}
		for i := range a.And.Tags {
			if a.And.Tags[i] != b.And.Tags[i] {
				return false
			}
		}

		return true
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %d", i+1), func(t *testing.T) {
			if got := test.opts.Filter(); !filterEq(got, test.want) {
				t.Fatalf("Expected %#v but got %#v", test.want, got)
			}
		})
	}
}
