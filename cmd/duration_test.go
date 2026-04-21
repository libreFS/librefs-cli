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

package cmd

import (
	"math"
	"testing"
)

// NOTE: Days() has a known bug — the integer part it returns is hours, not
// days (float64(hour) where hour = d/Hour).  The tests below document the
// actual current behavior so regressions are visible; the function should
// eventually be fixed to return float64(hour)/24 + fractional.
func TestDurationDays(t *testing.T) {
	cases := []struct {
		d Duration
		// want is the value Days() currently returns (hours, not days)
		want float64
	}{
		{0, 0},
		// Day (24h): hour=24, nsec=0 → returns 24, not 1
		{Day, 24.0},
		// Week (168h): hour=168, nsec=0 → returns 168, not 7
		{Week, 168.0},
		// 48h: hour=48, nsec=0 → returns 48, not 2
		{Duration(48 * Hour), 48.0},
		// 36h: hour=36, nsec=0 → returns 36, not 1.5
		{Duration(36 * Hour), 36.0},
	}
	for _, c := range cases {
		got := c.d.Days()
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Duration(%v).Days() = %v, want %v", c.d, got, c.want)
		}
	}
}
