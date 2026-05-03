// Copyright (c) 2015-2024 libreFS, Inc.
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

// helpers
func futureDate(daysFromNow int) lifecycle.ExpirationDate {
	t := time.Now().AddDate(0, 0, daysFromNow).Truncate(24 * time.Hour)
	return lifecycle.ExpirationDate{Time: t}
}

func futureDateStr(daysFromNow int) string {
	return time.Now().AddDate(0, 0, daysFromNow).Format(defaultILMDateFormat)
}

// ── extractILMTags ────────────────────────────────────────────────────────────

func TestExtractILMTags(t *testing.T) {
	cases := []struct {
		in   string
		want []lifecycle.Tag
	}{
		{"", nil},
		{"key1=val1", []lifecycle.Tag{{Key: "key1", Value: "val1"}}},
		{"key1=val1&key2=val2", []lifecycle.Tag{{Key: "key1", Value: "val1"}, {Key: "key2", Value: "val2"}}},
		// key with no value
		{"bare", []lifecycle.Tag{{Key: "bare", Value: ""}}},
	}
	for _, c := range cases {
		got := extractILMTags(c.in)
		if len(got) != len(c.want) {
			t.Errorf("extractILMTags(%q) len=%d, want %d", c.in, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("extractILMTags(%q)[%d] = %v, want %v", c.in, i, got[i], c.want[i])
			}
		}
	}
}

// ── validateTranDays ──────────────────────────────────────────────────────────

func TestValidateTranDays(t *testing.T) {
	cases := []struct {
		rule    lifecycle.Rule
		wantErr bool
	}{
		// negative days
		{lifecycle.Rule{Transition: lifecycle.Transition{Days: -1}}, true},
		// <30 days with STANDARD_IA
		{lifecycle.Rule{Transition: lifecycle.Transition{Days: 10, StorageClass: "STANDARD_IA"}}, true},
		// exactly 30 days with STANDARD_IA — ok
		{lifecycle.Rule{Transition: lifecycle.Transition{Days: 30, StorageClass: "STANDARD_IA"}}, false},
		// 0 days — ok
		{lifecycle.Rule{Transition: lifecycle.Transition{Days: 0}}, false},
		// positive days, non-IA class — ok
		{lifecycle.Rule{Transition: lifecycle.Transition{Days: 5, StorageClass: "GLACIER"}}, false},
	}
	for _, c := range cases {
		err := validateTranDays(c.rule)
		if (err != nil) != c.wantErr {
			t.Errorf("validateTranDays(%+v) err=%v, wantErr=%v", c.rule.Transition, err, c.wantErr)
		}
	}
}

// ── validateRuleAction ────────────────────────────────────────────────────────

func TestValidateRuleAction(t *testing.T) {
	// empty rule → error (no action set)
	if err := validateRuleAction(lifecycle.Rule{}); err == nil {
		t.Error("expected error for rule with no action")
	}
	// expiry days set → ok
	ruleWithExpiry := lifecycle.Rule{
		Expiration: lifecycle.Expiration{Days: 30},
	}
	if err := validateRuleAction(ruleWithExpiry); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// noncurrent expiry set → ok
	ruleNoncur := lifecycle.Rule{
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{NoncurrentDays: 10},
	}
	if err := validateRuleAction(ruleNoncur); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── validateExpiration ────────────────────────────────────────────────────────

func TestValidateExpiration(t *testing.T) {
	// only days — ok
	if err := validateExpiration(lifecycle.Rule{Expiration: lifecycle.Expiration{Days: 30}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// only date — ok
	if err := validateExpiration(lifecycle.Rule{Expiration: lifecycle.Expiration{Date: futureDate(10)}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// days + date — error (two params)
	if err := validateExpiration(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Days: 30, Date: futureDate(10)},
	}); err == nil {
		t.Error("expected error when both days and date are set")
	}
}

// ── validateTransition ────────────────────────────────────────────────────────

func TestValidateTransition(t *testing.T) {
	// only days — ok
	if err := validateTransition(lifecycle.Rule{Transition: lifecycle.Transition{Days: 30}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// only date — ok
	if err := validateTransition(lifecycle.Rule{Transition: lifecycle.Transition{Date: futureDate(10)}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// both — error
	if err := validateTransition(lifecycle.Rule{
		Transition: lifecycle.Transition{Days: 10, Date: futureDate(10)},
	}); err == nil {
		t.Error("expected error when both Transition days and date are set")
	}
}

// ── validateNoncurrentExpiration ──────────────────────────────────────────────

func TestValidateNoncurrentExpiration(t *testing.T) {
	if err := validateNoncurrentExpiration(lifecycle.Rule{
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{NoncurrentDays: -1},
	}); err == nil {
		t.Error("expected error for negative noncurrent expiration days")
	}
	if err := validateNoncurrentExpiration(lifecycle.Rule{
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{NoncurrentDays: 0},
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── validateNoncurrentTransition ─────────────────────────────────────────────

func TestValidateNoncurrentTransition(t *testing.T) {
	// negative days → error
	if err := validateNoncurrentTransition(lifecycle.Rule{
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{NoncurrentDays: -1},
	}); err == nil {
		t.Error("expected error for negative noncurrent transition days")
	}
	// days without storage class → error
	if err := validateNoncurrentTransition(lifecycle.Rule{
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{NoncurrentDays: 10, StorageClass: ""},
	}); err == nil {
		t.Error("expected error when days set without storage class")
	}
	// days + storage class → ok
	if err := validateNoncurrentTransition(lifecycle.Rule{
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{NoncurrentDays: 10, StorageClass: "GLACIER"},
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── validateTranExpDate ───────────────────────────────────────────────────────

func TestValidateTranExpDate(t *testing.T) {
	expDate := futureDate(20)
	tranDate := futureDate(10)
	// transition before expiry — ok
	if err := validateTranExpDate(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: expDate},
		Transition: lifecycle.Transition{Date: tranDate, StorageClass: "GLACIER"},
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// expiry before transition — error (tranDate is before expDate)
	if err := validateTranExpDate(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: tranDate},
		Transition: lifecycle.Transition{Date: expDate, StorageClass: "GLACIER"},
	}); err == nil {
		t.Error("expected error when expiry is before transition")
	}
	// no overlap (only expiry set, no transition) — ok
	if err := validateTranExpDate(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: expDate},
	}); err != nil {
		t.Errorf("unexpected error with expiry-only rule: %v", err)
	}
}

// ── validateTranExpCurdate ────────────────────────────────────────────────────

func TestValidateTranExpCurdate(t *testing.T) {
	// future expiry date — ok
	if err := validateTranExpCurdate(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: futureDate(10)},
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// past expiry date — error
	pastDate := lifecycle.ExpirationDate{Time: time.Now().AddDate(0, 0, -5)}
	if err := validateTranExpCurdate(lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: pastDate},
	}); err == nil {
		t.Error("expected error for past expiry date")
	}
	// future transition date — ok
	if err := validateTranExpCurdate(lifecycle.Rule{
		Transition: lifecycle.Transition{Date: futureDate(10), StorageClass: "GLACIER"},
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// past transition date — error
	if err := validateTranExpCurdate(lifecycle.Rule{
		Transition: lifecycle.Transition{Date: pastDate, StorageClass: "GLACIER"},
	}); err == nil {
		t.Error("expected error for past transition date")
	}
}

// ── parseTransitionDate / parseTransitionDays ─────────────────────────────────

func TestParseTransitionDate(t *testing.T) {
	s := futureDateStr(30)
	d, err := parseTransitionDate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.IsZero() {
		t.Error("parsed date should not be zero")
	}
	if _, err := parseTransitionDate("not-a-date"); err == nil {
		t.Error("expected error for invalid date string")
	}
}

func TestParseTransitionDays(t *testing.T) {
	d, err := parseTransitionDays("30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != lifecycle.ExpirationDays(30) {
		t.Errorf("got %d, want 30", d)
	}
	if _, err := parseTransitionDays("notanumber"); err == nil {
		t.Error("expected error for non-numeric input")
	}
}

// ── parseExpiryDate / parseExpiryDays ────────────────────────────────────────

func TestParseExpiryDate(t *testing.T) {
	s := futureDateStr(30)
	d, err := parseExpiryDate(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.IsZero() {
		t.Error("parsed date should not be zero")
	}
	if _, err := parseExpiryDate("bad-date"); err == nil {
		t.Error("expected error for invalid date string")
	}
}

func TestParseExpiryDays(t *testing.T) {
	d, err := parseExpiryDays("7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != lifecycle.ExpirationDays(7) {
		t.Errorf("got %d, want 7", d)
	}
	// zero → error
	if _, err := parseExpiryDays("0"); err == nil {
		t.Error("expected error for zero expiry days")
	}
	// non-numeric → error
	if _, err := parseExpiryDays("abc"); err == nil {
		t.Error("expected error for non-numeric input")
	}
}

// ── parseTransition ───────────────────────────────────────────────────────────

func TestParseTransition(t *testing.T) {
	sc := "GLACIER"
	days := "30"
	tr, err := parseTransition(&sc, nil, &days)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Days != lifecycle.ExpirationDays(30) {
		t.Errorf("Days = %d, want 30", tr.Days)
	}
	if tr.StorageClass != "GLACIER" {
		t.Errorf("StorageClass = %q, want GLACIER", tr.StorageClass)
	}
	// nil everything → empty transition
	tr2, err := parseTransition(nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr2.StorageClass != "" || tr2.Days != 0 {
		t.Errorf("expected empty transition, got %+v", tr2)
	}
	// invalid days → error
	bad := "notanumber"
	if _, err := parseTransition(nil, nil, &bad); err == nil {
		t.Error("expected error for invalid days")
	}
}

// ── parseExpiry ───────────────────────────────────────────────────────────────

func TestParseExpiry(t *testing.T) {
	days := "14"
	exp, err := parseExpiry(nil, &days, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.Days != lifecycle.ExpirationDays(14) {
		t.Errorf("Days = %d, want 14", exp.Days)
	}

	dateStr := futureDateStr(30)
	exp2, err := parseExpiry(&dateStr, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp2.Date.IsZero() {
		t.Error("expected non-zero date")
	}

	tr := true
	exp3, err := parseExpiry(nil, nil, &tr, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bool(exp3.DeleteMarker) {
		t.Error("expected DeleteMarker to be true")
	}

	// zero expiry days → error propagated
	zero := "0"
	if _, err := parseExpiry(nil, &zero, nil, nil); err == nil {
		t.Error("expected error for zero expiry days")
	}
}
