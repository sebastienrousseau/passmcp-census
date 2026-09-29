// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package edition

import "testing"

func TestValid(t *testing.T) {
	for id, ok := range map[string]bool{
		"2026-09": true, "2027-12": true, "2026-13": false, "2026-00": false,
		"2026-9": false, "26-09": false, "": false, "../etc": false,
	} {
		if err := Valid(id); (err == nil) != ok {
			t.Errorf("Valid(%q) = %v", id, err)
		}
	}
}
