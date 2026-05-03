// Copyright (c) 2022 libreFS, Inc.
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
	"testing"
	"time"

	"github.com/libreFS/librefs-go/v7/pkg/lifecycle"
)

func TestILMTags(t *testing.T) {
	tests := []struct {
		rule     lifecycle.Rule
		expected string
	}{
		{
			rule: lifecycle.Rule{
				ID: "one-tag",
				RuleFilter: lifecycle.Filter{
					Tag: lifecycle.Tag{
						Key:   "key1",
						Value: "val1",
					},
				},
			},
			expected: "key1=val1",
		},
		{
			rule: lifecycle.Rule{
				ID: "many-tags",
				RuleFilter: lifecycle.Filter{
					And: lifecycle.And{
						Tags: []lifecycle.Tag{
							{
								Key:   "key1",
								Value: "val1",
							},
							{
								Key:   "key2",
								Value: "val2",
							},
							{
								Key:   "key3",
								Value: "val3",
							},
						},
					},
				},
			},
			expected: "key1=val1&key2=val2&key3=val3",
		},
	}
	for i, test := range tests {
		if got := getTags(test.rule); got != test.expected {
			t.Fatalf("%d: Expected %s but got %s", i+1, test.expected, got)
		}
	}
}

// ── getPrefix ─────────────────────────────────────────────────────────────────

func TestGetPrefix(t *testing.T) {
	cases := []struct {
		rule lifecycle.Rule
		want string
	}{
		// deprecated top-level prefix
		{lifecycle.Rule{Prefix: "logs/"}, "logs/"},
		// filter prefix
		{lifecycle.Rule{RuleFilter: lifecycle.Filter{Prefix: "docs/"}}, "docs/"},
		// And prefix
		{lifecycle.Rule{RuleFilter: lifecycle.Filter{And: lifecycle.And{Prefix: "data/"}}}, "data/"},
		// nothing set
		{lifecycle.Rule{}, ""},
	}
	for _, c := range cases {
		got := getPrefix(c.rule)
		if got != c.want {
			t.Errorf("getPrefix(%+v) = %q, want %q", c.rule, got, c.want)
		}
	}
}

// ── getExpirationDays ─────────────────────────────────────────────────────────

func TestGetExpirationDays(t *testing.T) {
	// explicit days
	r := lifecycle.Rule{Expiration: lifecycle.Expiration{Days: 30}}
	if got := getExpirationDays(r); got != 30 {
		t.Errorf("getExpirationDays(Days=30) = %d, want 30", got)
	}
	// no expiration → 0
	if got := getExpirationDays(lifecycle.Rule{}); got != 0 {
		t.Errorf("getExpirationDays(empty) = %d, want 0", got)
	}
	// date in the future → positive days
	future := lifecycle.ExpirationDate{Time: time.Now().Add(10 * 24 * time.Hour)}
	r2 := lifecycle.Rule{Expiration: lifecycle.Expiration{Date: future}}
	if got := getExpirationDays(r2); got <= 0 {
		t.Errorf("getExpirationDays(future date) = %d, want >0", got)
	}
}

// ── getTransitionDays ─────────────────────────────────────────────────────────

func TestGetTransitionDays(t *testing.T) {
	// explicit days
	r := lifecycle.Rule{Transition: lifecycle.Transition{Days: 45}}
	if got := getTransitionDays(r); got != 45 {
		t.Errorf("getTransitionDays(Days=45) = %d, want 45", got)
	}
	// no transition → 0
	if got := getTransitionDays(lifecycle.Rule{}); got != 0 {
		t.Errorf("getTransitionDays(empty) = %d, want 0", got)
	}
	// date in the future → positive days
	future := lifecycle.ExpirationDate{Time: time.Now().Add(10 * 24 * time.Hour)}
	r2 := lifecycle.Rule{Transition: lifecycle.Transition{Date: future}}
	if got := getTransitionDays(r2); got <= 0 {
		t.Errorf("getTransitionDays(future date) = %d, want >0", got)
	}
}

// ── ToTables ──────────────────────────────────────────────────────────────────

func TestToTables(t *testing.T) {
	// empty config → no tables
	if tables := ToTables(&lifecycle.Configuration{}); len(tables) != 0 {
		t.Errorf("ToTables(empty) = %d tables, want 0", len(tables))
	}

	// rule with expiry → one table
	cfg := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				ID:         "expire-rule",
				Status:     "Enabled",
				Expiration: lifecycle.Expiration{Days: 30},
			},
		},
	}
	tables := ToTables(cfg)
	if len(tables) != 1 {
		t.Fatalf("ToTables(one expiry rule) = %d tables, want 1", len(tables))
	}
	if tables[0].Len() != 1 {
		t.Errorf("table Len() = %d, want 1", tables[0].Len())
	}

	// rule with transition → separate table for tier
	cfg2 := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				ID:         "tier-rule",
				Status:     "Enabled",
				Transition: lifecycle.Transition{Days: 60, StorageClass: "GLACIER"},
			},
		},
	}
	tables2 := ToTables(cfg2)
	if len(tables2) != 1 {
		t.Fatalf("ToTables(one transition rule) = %d tables, want 1", len(tables2))
	}

	// rule with both expiry and transition → two tables
	cfg3 := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				ID:         "both-rule",
				Status:     "Enabled",
				Expiration: lifecycle.Expiration{Days: 90},
				Transition: lifecycle.Transition{Days: 30, StorageClass: "GLACIER"},
			},
		},
	}
	tables3 := ToTables(cfg3)
	if len(tables3) != 2 {
		t.Fatalf("ToTables(expiry+transition rule) = %d tables, want 2", len(tables3))
	}
}
